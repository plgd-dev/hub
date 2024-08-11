//go:build test_mem
// +build test_mem

package service_test

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	coapService "github.com/plgd-dev/hub/v2/coap-gateway/service"
	coapgwTest "github.com/plgd-dev/hub/v2/coap-gateway/test"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	rdTest "github.com/plgd-dev/hub/v2/resource-directory/test"
	test "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/service"
	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type testingT struct {
	err error
}

func (t *testingT) Errorf(format string, args ...interface{}) {
	t.err = fmt.Errorf(format, args...)
}

func (t *testingT) FailNow() {
	panic(t.err)
}

func checkServices() bool {
	defer func() {
		_ = recover()
	}()

	var nt testingT
	code := oauthTest.GetDefaultAccessToken(&nt)
	if nt.err == nil && code != "" {
		return true
	}
	return false
}

func shell(name string, format string, arg ...any) {
	cmd := fmt.Sprintf(format, arg...)
	out, err := exec.Command("/bin/bash", "-c", cmd).CombinedOutput()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s %s", name, out)
}

// PrintMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func PrintMemUsage(started time.Time) {
	rss := uint64(0)
	out, err := exec.Command("ps", "-h", "-o", "rss", strconv.Itoa(os.Getpid())).Output()
	if err == nil {
		if len(out) > 0 && out[len(out)-1] == '\n' {
			out = out[:len(out)-1]
		}
		rss, _ = strconv.ParseUint(string(out), 10, 64)
	} else {
		fmt.Println(err)
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Sys = %v MiB, RSS = %v MiB PID = %v\n", bToMb(m.Sys), rss/1024, os.Getpid())
	fmt.Printf("%v ---RAM---\n", time.Since(started))
	pid := os.Getpid()
	shell("ps rss:  ", `ps -o rss= -p %d`, pid)
	shell("top res: ", `top -b -n 1 | awk '/^[ ]*%d[ ]/ {print $6}'`, pid)
	shell("LazyFree:", `cat /proc/%d/smaps | awk '/LazyFree/ {sum+= $2} END {print sum}'`, pid)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func GetMemRSS(pid int) uint64 {
	rss := uint64(0)
	out, err := exec.Command("ps", "-h", "-o", "rss", strconv.Itoa(pid)).Output()
	if err == nil {
		if len(out) > 0 && out[len(out)-1] == '\n' {
			out = out[:len(out)-1]
		}
		rss, _ = strconv.ParseUint(string(out), 10, 64)
	}
	return rss * 1024
}

// ByteSlice2String converts a byte slice to a string without memory allocation.
func ByteSlice2String(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

/*
//go:noinline
func logTest(logger log.Logger, size int) {
	buffer := make([]byte, size)
	for i := 0; i < size; i++ {
		buffer[i] = 'a'
	}
	logger = logger.With("test", ByteSlice2String(buffer))
	logger.Debug("test")
}

func testAllocation(t *testing.T) {
	race := israce.Enabled
	bufferSize := 1024 * 1024 * 10
	cfg := log.MakeDefaultConfig()
	logger := log.NewLogger(cfg)

	rssMb := bToMb(GetMemRSS())
	t.Log("before test - mem rss: ", rssMb, "MB")
	for i := 0; i < 10; i++ {
		logTest(logger, bufferSize)
	}
	debug.FreeOSMemory()
	debug.FreeOSMemory()

	rssMb = bToMb(GetMemRSS())
	t.Log("after mem rss: ", rssMb, "MB")
	if race {
		require.Less(t, rssMb, uint64(500))
	} else {
		require.Less(t, rssMb, uint64(33))
	}
}
*/

func TestMemoryWithDevices(t *testing.T) {
	numDevices, err := strconv.Atoi(os.Getenv("TEST_MEMORY_COAP_GATEWAY_NUM_DEVICES"))
	require.NoError(t, err)
	numResources, err := strconv.Atoi(os.Getenv("TEST_MEMORY_COAP_GATEWAY_NUM_RESOURCES"))
	require.NoError(t, err)
	expRSSInMB, err := strconv.Atoi(os.Getenv("TEST_MEMORY_COAP_GATEWAY_EXPECTED_RSS_IN_MB"))
	require.NoError(t, err)
	resourceDataSize, err := strconv.Atoi(os.Getenv("TEST_MEMORY_COAP_GATEWAY_RESOURCE_DATA_SIZE"))
	require.NoError(t, err)
	testDevices(t, numDevices, numResources, expRSSInMB, resourceDataSize)
}

func testDevices(t *testing.T, numDevices, numResources, expRSSInMB int, resourceDataSize int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// build services
	servicesPath := "./services"
	err := exec.CommandContext(ctx, "go", "build", "-o", servicesPath, "github.com/plgd-dev/hub/v2/test/service/cmd").Run()
	require.NoError(t, err)

	// build virtual device
	vdPath := "./vd"
	err = exec.CommandContext(ctx, "go", "build", "-o", vdPath, "github.com/plgd-dev/hub/v2/test/virtual-device/cmd").Run()
	require.NoError(t, err)
	rdConfig := rdTest.MakeConfig(&testingT{})
	rdConfig.Clients.Eventstore.ProjectionCacheExpiration = time.Second * 2
	const services = service.SetUpServicesOAuth | service.SetUpServicesMachine2MachineOAuth | service.SetUpServicesId | service.SetUpServicesResourceDirectory |
		service.SetUpServicesGrpcGateway | service.SetUpServicesResourceAggregate
	rdCfgFile, err := os.CreateTemp("", "rd")
	require.NoError(t, err)
	rdCfgPath := rdCfgFile.Name()
	_, err = rdCfgFile.WriteString(rdConfig.String())
	require.NoError(t, err)
	err = rdCfgFile.Sync()
	require.NoError(t, err)
	err = rdCfgFile.Close()
	require.NoError(t, err)
	defer func() {
		err := os.Remove(rdCfgPath)
		require.NoError(t, err)
	}()
	cmdService := exec.CommandContext(ctx, servicesPath, "--services", strconv.Itoa(int(services)), "--rdConfig", rdCfgPath)
	cmdService.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmdService.Start()
	require.NoError(t, err)

	defer func() {
		err := syscall.Kill(cmdService.Process.Pid, syscall.SIGKILL)
		require.NoError(t, err)
		_, err = cmdService.Process.Wait()
		require.NoError(t, err)
	}()

	now := time.Now()
	maxTries := 30
	for {
		time.Sleep(time.Second * 2)
		if checkServices() {
			break
		}
		serviceRSS := GetMemRSS(cmdService.Process.Pid)
		if serviceRSS == 0 {
			_, err = cmdService.Process.Wait()
			require.NoError(t, err)
			if maxTries != 0 {
				cmdService.Process = nil
				err = cmdService.Start()
				require.NoError(t, err)
			} else {
				require.NoError(t, errors.New("service process is dead - canceling test"))
			}
		}
		t.Logf("waiting for service to start: %v, serviceRSS %v\n", time.Since(now), bToMb(serviceRSS))
		maxTries--
	}

	ctx = kitNetGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	cfg := coapgwTest.MakeConfig(t)
	cfg.APIs.COAP.MaxMessageSize = uint32(numResources*(resourceDataSize+1024) + 8*1024)
	cfg.APIs.COAP.BlockwiseTransfer.Enabled = true

	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	s, err := coapService.New(ctx, cfg, fileWatcher, logger)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = s.Serve()
	}()

	coapShutdown := func() {
		errC := s.Close()
		require.NoError(t, errC)
		wg.Wait()
	}
	defer coapShutdown()
	grpcConn, err := grpc.NewClient(config.GRPC_GW_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: test.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	client := pb.NewGrpcGatewayClient(grpcConn)
	subClient, err := client.SubscribeToEvents(ctx)
	require.NoError(t, err)
	err = subClient.Send(&pb.SubscribeToEvents{
		Action: &pb.SubscribeToEvents_CreateSubscription_{
			CreateSubscription: &pb.SubscribeToEvents_CreateSubscription{
				EventFilter: []pb.SubscribeToEvents_CreateSubscription_Event{
					pb.SubscribeToEvents_CreateSubscription_DEVICE_METADATA_UPDATED,
					pb.SubscribeToEvents_CreateSubscription_REGISTERED,
				},
			},
		},
	})
	require.NoError(t, err)
	resp, err := subClient.Recv()
	require.NoError(t, err)
	require.Equal(t, pb.Event_OperationProcessed_ErrorStatus_OK, resp.GetOperationProcessed().GetErrorStatus().GetCode())

	beforeTestRSSMB := bToMb(GetMemRSS(os.Getpid()))
	startTime := time.Now()

	cmdVD := exec.CommandContext(ctx, vdPath, "--numDevices", strconv.Itoa(numDevices), "--numResources", strconv.Itoa(numResources), "--resourceDataSize", strconv.Itoa(resourceDataSize))
	cmdVD.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmdVD.Start()
	require.NoError(t, err)
	defer func() {
		err := syscall.Kill(-cmdVD.Process.Pid, syscall.SIGKILL)
		require.NoError(t, err)
		_, err = cmdVD.Process.Wait()
		require.NoError(t, err)
	}()

	go func(vdPid, servicePid int) {
		ticker := time.NewTicker(time.Second * 30)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				vdRSS := GetMemRSS(vdPid)
				if vdRSS == 0 {
					t.Logf("virtual device is dead - canceling test")
					cancel()
					return
				}
				serviceRSS := GetMemRSS(servicePid)
				if serviceRSS == 0 {
					t.Logf("service process is dead - canceling test")
					cancel()
					return
				}
				t.Logf("vdRSS=%v, serviceRSS=%v", bToMb(vdRSS), bToMb(serviceRSS))
			}
		}
	}(cmdVD.Process.Pid, cmdService.Process.Pid)

	syncedDevices := 0
	for {
		resp, err := subClient.Recv()
		require.NoError(t, err)

		if resp.GetDeviceMetadataUpdated().GetTwinSynchronization().GetState() == commands.TwinSynchronization_IN_SYNC {
			syncedDevices++
			if syncedDevices == numDevices {
				break
			}
		}

		if resp.GetDeviceRegistered() != nil {
			t.Logf("devices registered %v", resp.GetDeviceRegistered().GetDeviceIds())
		}
	}
	_ = grpcConn.Close()

	debug.FreeOSMemory()
	debug.FreeOSMemory()
	rssMb := bToMb(GetMemRSS(os.Getpid()))
	duration := time.Since(startTime)
	v := struct {
		NumDevices       int
		NumResources     int
		ExpectedMemRSS   int
		CurrentMemRSS    int
		InitMemRSS       int
		ResourceDataSize int
		LogLevel         string
		LogDumpBody      bool
		Duration         time.Duration
	}{
		NumDevices:       numDevices,
		NumResources:     numResources,
		ExpectedMemRSS:   expRSSInMB,
		CurrentMemRSS:    int(rssMb),
		InitMemRSS:       int(beforeTestRSSMB),
		LogLevel:         cfg.Log.Level.String(),
		LogDumpBody:      cfg.Log.DumpBody,
		Duration:         duration,
		ResourceDataSize: resourceDataSize,
	}
	data, err := json.Encode(v)
	require.NoError(t, err)

	t.Logf("TestMemoryWithDevices.result:%s", data)
	strErr := fmt.Sprintf("memory usage is too high for numDevices %v, numResources %v", numDevices, numResources)
	assert.Less(t, rssMb, uint64(expRSSInMB) /*MB*/, strErr)
}
