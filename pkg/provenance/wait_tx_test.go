package provenance

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type fakeTxService struct {
	txtypes.UnimplementedServiceServer
	mu      sync.Mutex
	hits    int
	readyAt int
	hash    string
}

func (s *fakeTxService) GetTx(_ context.Context, req *txtypes.GetTxRequest) (*txtypes.GetTxResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hits++
	if req.Hash != s.hash || s.hits < s.readyAt {
		return nil, status.Error(codes.NotFound, "tx not found")
	}
	return &txtypes.GetTxResponse{TxResponse: &sdk.TxResponse{TxHash: s.hash, Height: 42, Code: 0}}, nil
}

func testClientWithTxServer(t *testing.T, svc txtypes.ServiceServer) *ProvenanceClient {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	txtypes.RegisterServiceServer(srv, svc)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return &ProvenanceClient{Grpc: &GRPCConnection{Conn: conn}}
}

func TestWaitOnTxRetriesUntilIndexed(t *testing.T) {
	const hash = "ABCDEF"
	c := testClientWithTxServer(t, &fakeTxService{readyAt: 3, hash: hash})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := c.waitOnTx(ctx, hash, 5*time.Second, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if got.TxResponse == nil || got.TxResponse.TxHash != hash || got.TxResponse.Height != 42 {
		t.Fatalf("response %+v", got.TxResponse)
	}
}

func TestWaitOnTxTimeoutIncludesLastError(t *testing.T) {
	const hash = "DEADBEEF"
	c := testClientWithTxServer(t, &fakeTxService{readyAt: 100, hash: hash})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.waitOnTx(ctx, hash, 80*time.Millisecond, 10*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), hash) || !strings.Contains(err.Error(), "NotFound") {
		t.Fatalf("got %v", err)
	}
}

func TestWaitOnTxHonorsCancel(t *testing.T) {
	c := testClientWithTxServer(t, &fakeTxService{readyAt: 100, hash: "ABC"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.waitOnTx(ctx, "ABC", time.Minute, time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestWaitOnTxRejectsEmptyHash(t *testing.T) {
	_, err := (&ProvenanceClient{}).WaitOnTx("")
	if err == nil || !strings.Contains(err.Error(), "no tx hash") {
		t.Fatalf("got %v", err)
	}
}

// slowTxService waits, then returns a successful tx. It ignores cancellation so a client
// that does not bound GetTx observes a late success.
type slowTxService struct {
	txtypes.UnimplementedServiceServer
	delay time.Duration
}

func (s *slowTxService) GetTx(_ context.Context, req *txtypes.GetTxRequest) (*txtypes.GetTxResponse, error) {
	time.Sleep(s.delay)
	return &txtypes.GetTxResponse{TxResponse: &sdk.TxResponse{TxHash: req.Hash, Height: 9, Code: 0}}, nil
}

func waitConnReady(t *testing.T, conn *grpc.ClientConn) {
	t.Helper()
	conn.Connect()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return
		}
		if !conn.WaitForStateChange(ctx, state) {
			t.Fatalf("grpc conn not ready: %s", conn.GetState())
		}
	}
}

func TestWaitOnTxStalledRPCStopsAtBudget(t *testing.T) {
	const hash = "STALL"
	c := testClientWithTxServer(t, &slowTxService{delay: 400 * time.Millisecond})
	waitConnReady(t, c.Grpc.Conn)

	start := time.Now()
	_, err := c.waitOnTx(context.Background(), hash, 30*time.Millisecond, time.Second)
	elapsed := time.Since(start)
	if err == nil || !strings.Contains(err.Error(), hash) {
		t.Fatalf("got %v after %s", err, elapsed)
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("stalled GetTx returned after %s, want the 30ms budget", elapsed)
	}
}
