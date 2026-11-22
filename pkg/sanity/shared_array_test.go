/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sanity

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	"github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Run the registered sanity specs with an in-memory CSI responder. Ginkgo
// can run only once per process; inspect its per-spec report, including
// deliberate failures, so a skipped spec or a fixture error cannot pass.
func TestSharedArrayInventory(t *testing.T) {
	const volume = "check the presence of new volumes"
	const snapshot = "check the presence of new snapshots"
	const pages = "should return next token when a limited number"
	const all = volume + "|" + snapshot + "|" + pages
	for _, tc := range []struct {
		mode, fault, focus, failure string
		count                       int
	}{
		{"dedicated", "none", all, "", 3},
		{"shared", "none", all, "", 3},
		{"dedicated", "concurrent", volume, "dedicated array volume count", 1},
		{"dedicated", "concurrent", snapshot, "dedicated array snapshot count", 1},
		{"shared", "concurrent", volume + "|" + snapshot, "", 2},
		{"shared", "hide-created", volume, "created volume is missing", 1},
		{"shared", "hide-created", snapshot, "created snapshot is missing", 1},
		{"shared", "retain-deleted", volume, "deleted volume is still listed", 1},
		{"shared", "retain-deleted", snapshot, "deleted snapshot is still listed", 1},
		{"shared", "duplicate", pages, "paginated snapshot IDs differ", 1},
		{"shared", "omit", pages, "paginated snapshot IDs differ", 1},
	} {
		t.Run(tc.mode+"/"+tc.fault+"/"+tc.focus, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "report.json")
			cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestInventoryResponder$", "-ginkgo.focus="+tc.focus, "-ginkgo.json-report="+file, "-ginkgo.no-color")
			cmd.Env = append(os.Environ(), "CSI_TEST_FIXTURE_MODE="+tc.mode, "CSI_TEST_FIXTURE_FAULT="+tc.fault)
			out, err := cmd.CombinedOutput()
			if (err != nil) != (tc.failure != "") {
				t.Fatalf("unexpected child exit %v\n%s", err, out)
			}
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var reports []types.Report
			if err = json.Unmarshal(raw, &reports); err != nil {
				t.Fatal(err)
			}
			if len(reports) != 1 {
				t.Fatalf("got %d suite reports", len(reports))
			}
			count := 0
			for _, spec := range reports[0].SpecReports {
				if spec.State == types.SpecStateSkipped || spec.State == types.SpecStatePending {
					continue
				}
				count++
				if tc.failure == "" && spec.State == types.SpecStatePassed {
					continue
				}
				if tc.failure != "" && spec.State == types.SpecStateFailed && strings.Contains(spec.Failure.Message, tc.failure) {
					continue
				}
				t.Fatalf("%s: %s: %s", spec.LeafNodeText, spec.State, spec.Failure.Message)
			}
			if count != tc.count {
				t.Fatalf("ran %d specs, want %d", count, tc.count)
			}
		})
	}
}

func TestInventoryResponder(t *testing.T) {
	mode := os.Getenv("CSI_TEST_FIXTURE_MODE")
	if mode == "" {
		return
	}
	cfg := NewTestConfig()
	if mode == "shared" {
		cfg.SharedArray = true
	}
	cfg.Address = "passthrough:///inventory"
	cfg.TargetPath, cfg.StagingPath = "", ""
	fixture := inventoryResponder{t: t, shared: mode == "shared", fault: os.Getenv("CSI_TEST_FIXTURE_FAULT"), volumes: map[string]bool{}, snapshots: map[string]string{"existing": "another-volume"}}
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithUnaryInterceptor(fixture.respond))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	sc := GinkgoTest(&cfg)
	sc.Conn, sc.connAddress = conn, cfg.Address
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Shared array inventory")
}

type inventoryResponder struct {
	t         *testing.T
	shared    bool
	fault     string
	volumes   map[string]bool
	snapshots map[string]string
}

func (f *inventoryResponder) respond(_ context.Context, method string, req, res any, _ *grpc.ClientConn, _ grpc.UnaryInvoker, _ ...grpc.CallOption) error {
	switch method[strings.LastIndexByte(method, '/')+1:] {
	case "ControllerGetCapabilities":
		r := res.(*csi.ControllerGetCapabilitiesResponse)
		for _, capability := range []csi.ControllerServiceCapability_RPC_Type{csi.ControllerServiceCapability_RPC_LIST_VOLUMES, csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS} {
			r.Capabilities = append(r.Capabilities, &csi.ControllerServiceCapability{Type: &csi.ControllerServiceCapability_Rpc{Rpc: &csi.ControllerServiceCapability_RPC{Type: capability}}})
		}
	case "NodeGetCapabilities", "NodeUnpublishVolume":
	case "CreateVolume":
		id := req.(*csi.CreateVolumeRequest).Name
		f.volumes[id] = true
		res.(*csi.CreateVolumeResponse).Volume = &csi.Volume{VolumeId: id}
	case "DeleteVolume":
		if f.fault != "retain-deleted" {
			delete(f.volumes, req.(*csi.DeleteVolumeRequest).VolumeId)
		}
	case "ListVolumes":
		r := res.(*csi.ListVolumesResponse)
		for id := range f.volumes {
			if f.fault != "hide-created" {
				r.Entries = append(r.Entries, &csi.ListVolumesResponse_Entry{Volume: &csi.Volume{VolumeId: id}})
			}
		}
		if f.fault == "concurrent" && len(f.volumes) > 0 {
			r.Entries = append(r.Entries, &csi.ListVolumesResponse_Entry{Volume: &csi.Volume{VolumeId: "foreign"}})
		}
	case "CreateSnapshot":
		q := req.(*csi.CreateSnapshotRequest)
		f.snapshots[q.Name] = q.SourceVolumeId
		res.(*csi.CreateSnapshotResponse).Snapshot = &csi.Snapshot{SnapshotId: q.Name, SourceVolumeId: q.SourceVolumeId, CreationTime: timestamppb.Now()}
	case "DeleteSnapshot":
		id := req.(*csi.DeleteSnapshotRequest).SnapshotId
		if id == "existing" {
			f.t.Error("deleted the pre-existing snapshot")
		}
		if f.fault != "retain-deleted" {
			delete(f.snapshots, id)
		}
	case "ListSnapshots":
		f.listSnapshots(req.(*csi.ListSnapshotsRequest), res.(*csi.ListSnapshotsResponse))
	default:
		f.t.Errorf("unexpected RPC: %s", method)
	}
	return nil
}

func (f *inventoryResponder) listSnapshots(q *csi.ListSnapshotsRequest, r *csi.ListSnapshotsResponse) {
	var ids []string
	for id, volume := range f.snapshots {
		if (q.SourceVolumeId == "" || q.SourceVolumeId == volume) && (f.fault != "hide-created" || id == "existing") {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	if f.fault == "concurrent" && len(f.snapshots) > 1 {
		ids = append(ids, "foreign")
	}
	if q.MaxEntries != 0 || q.StartingToken != "" {
		if f.shared && !f.volumes[q.SourceVolumeId] || !f.shared && q.SourceVolumeId != "" {
			f.t.Error("wrong source volume on pagination request")
		}
		if len(ids) != 5 {
			f.t.Fatalf("expected five snapshots in pagination scope, got %d", len(ids))
		}
		if q.StartingToken == "" {
			if q.MaxEntries != 2 {
				f.t.Fatalf("unexpected page size %d", q.MaxEntries)
			}
			ids = ids[:2]
			r.NextToken = "second"
		} else {
			if q.StartingToken != "second" {
				f.t.Fatal("wrong continuation token")
			}
			ids = ids[2:]
			switch f.fault {
			case "duplicate":
				ids[1] = ids[0]
			case "omit":
				ids = ids[1:]
			}
		}
	}
	for _, id := range ids {
		r.Entries = append(r.Entries, &csi.ListSnapshotsResponse_Entry{Snapshot: &csi.Snapshot{SnapshotId: id, SourceVolumeId: f.snapshots[id]}})
	}
}
