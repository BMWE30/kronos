package kronos

import (
	"context"
	"errors"
	"github.com/gogo/protobuf/proto"
	"time"

	"github.com/rubrikinc/kronos/kronoshttp"
	"github.com/rubrikinc/kronos/kronosstats"
	"github.com/rubrikinc/kronos/kronosutil/log"
	"github.com/BMWE30/kronos/server"
)

//var kronosServer *server.Server

// Initialize initializes the kronos server.
// After Initialization, Now() in this package returns kronos time.
// If not initialized, Now() in this package returns system time
func Initialize(ctx context.Context, config server.Config, selfID string) (*server.Server, error) {
	// Stop previous server
	//if kronosServer != nil {
	//	kronosServer.Stop()
	//}

	var err error
	kronosServer, err := server.NewKronosServer(ctx, config, selfID)
	if err != nil {
		return nil, err
	}

	go func() {
		if err := kronosServer.RunServer(ctx); err != nil {
			log.Fatal(ctx, err)
		}
	}()

	log.Info(ctx, "Kronos server initialized")
	return kronosServer, nil
}

// Stop stops the kronos server
func Stop(kronosServer *server.Server) {
	if kronosServer != nil {
		kronosServer.Stop()
		log.Info(context.TODO(), "Kronos server stopped")
	}
}

// IsActive returns whether kronos is running.
func IsActive(kronosServer *server.Server) bool {
	return kronosServer != nil
}

func IsOracle(ctx context.Context, kronosServer *server.Server) bool {
	return proto.Equal(kronosServer.GRPCAddr, kronosServer.OracleSM.State(ctx).Oracle)
}

// Now returns Kronos time if Kronos is initialized, otherwise returns
// system time
func Now(kronosServer *server.Server) int64 {
	if kronosServer == nil {
		log.Fatalf(context.TODO(), "Kronos server is not initialized")
	}
	// timePollInterval is the time to wait before internally retrying
	// this function.
	// This function blocks if not initialized or if KronosTime is stale
	const timePollInterval = 100 * time.Millisecond
	ctx := context.TODO()

	for {
		kt, err := kronosServer.KronosTimeNow(ctx)
		if err != nil {
			log.Errorf(
				ctx,
				"Failed to get KronosTime, err: %v. Sleeping for %s before retrying.",
				err, timePollInterval,
			)
			time.Sleep(timePollInterval)
			continue
		}
		return kt.Time
	}
}

// NodeID returns the NodeID of the kronos server in the kronos raft cluster.
// NodeID returns an empty string if kronosServer is not initialized
func NodeID(ctx context.Context, kronosServer *server.Server) string {
	if kronosServer == nil {
		return ""
	}

	id, err := kronosServer.ID()
	if err != nil {
		log.Fatalf(ctx, "Failed to get kronosServer.ID, err: %v", err)
	}

	return id
}

func GetID(kronosServer *server.Server) string{
	return kronosServer.GetID()
}

func AddNode(kronosServer *server.Server, id, addr string){
	kronosServer.AddNode(id, addr)
}

// RemoveNode removes the given node from the kronos raft cluster
func RemoveNode(ctx context.Context, nodeID string, kronosServer *server.Server) error {
	if len(nodeID) == 0 {
		return errors.New("node id is empty")
	}

	log.Infof(ctx, "Removing kronos node %s", nodeID)
	client, err := kronosServer.NewClusterClient()
	if err != nil {
		return err
	}
	defer client.Close()

	return client.RemoveNode(ctx, &kronoshttp.RemoveNodeRequest{
		NodeID: nodeID,
	})
}

// Metrics returns KronosMetrics
func Metrics(kronosServer *server.Server) *kronosstats.KronosMetrics {
	if kronosServer == nil {
		return nil
	}
	return kronosServer.Metrics
}

func GenerateNewNodeID() string{
	return server.GenerateNewNodeID()
}
