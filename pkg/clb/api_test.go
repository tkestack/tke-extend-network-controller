package clb

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
)

// mockResourceInOperatingError 构造 ResourceInOperating 错误
func mockResourceInOperatingError() error {
	return errors.New("[TencentCloudSDKError] Code=FailedOperation.ResourceInOperating, Message=Loadbalancer(lb-test) is not in normal desState, maybe some tasks are being processed, please try later.")
}

func TestApiCallRetryOnResourceInOperating(t *testing.T) {
	var callCount atomic.Int32
	ctx := context.Background()

	// 前 2 次返回 ResourceInOperating，第 3 次成功
	_, err := ApiCall(ctx, true, "BatchRegisterTargets", "ap-test", func(ctx context.Context, client *clb.Client) (req *clb.BatchRegisterTargetsRequest, res *clb.BatchRegisterTargetsResponse, err error) {
		n := callCount.Add(1)
		req = clb.NewBatchRegisterTargetsRequest()
		if n <= 2 {
			return req, nil, mockResourceInOperatingError()
		}
		res = &clb.BatchRegisterTargetsResponse{}
		return req, res, nil
	})

	if err != nil {
		t.Fatalf("expected retry to succeed, got error: %v", err)
	}
	if callCount.Load() != 3 {
		t.Fatalf("expected 3 calls (2 retries + 1 success), got %d", callCount.Load())
	}
}

func TestApiCallGiveUpAfterMaxRetries(t *testing.T) {
	var callCount atomic.Int32
	ctx := context.Background()

	// 一直返回 ResourceInOperating，超过最大重试次数后返回错误
	_, err := ApiCall(ctx, true, "BatchRegisterTargets", "ap-test", func(ctx context.Context, client *clb.Client) (req *clb.BatchRegisterTargetsRequest, res *clb.BatchRegisterTargetsResponse, err error) {
		callCount.Add(1)
		req = clb.NewBatchRegisterTargetsRequest()
		return req, nil, mockResourceInOperatingError()
	})

	if err == nil {
		t.Fatal("expected error after max retries")
	}
	if !strings.Contains(err.Error(), "ResourceInOperating") {
		t.Fatalf("expected ResourceInOperating error, got: %v", err)
	}
	// 1 次初始调用 + 5 次重试
	if callCount.Load() != resourceInOperatingRetryMax+1 {
		t.Fatalf("expected %d calls, got %d", resourceInOperatingRetryMax+1, callCount.Load())
	}
}

func TestApiCallNoRetryOnReadOp(t *testing.T) {
	var callCount atomic.Int32
	ctx := context.Background()

	// 读操作不重试 ResourceInOperating
	_, err := ApiCall(ctx, false, "DescribeListeners", "ap-test", func(ctx context.Context, client *clb.Client) (req *clb.DescribeListenersRequest, res *clb.DescribeListenersResponse, err error) {
		callCount.Add(1)
		req = clb.NewDescribeListenersRequest()
		return req, nil, mockResourceInOperatingError()
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if callCount.Load() != 1 {
		t.Fatalf("expected 1 call (no retry for read op), got %d", callCount.Load())
	}
}

func TestApiCallNoRetryOnOtherError(t *testing.T) {
	var callCount atomic.Int32
	ctx := context.Background()

	// 非 ResourceInOperating 错误不重试
	_, err := ApiCall(ctx, true, "BatchRegisterTargets", "ap-test", func(ctx context.Context, client *clb.Client) (req *clb.BatchRegisterTargetsRequest, res *clb.BatchRegisterTargetsResponse, err error) {
		callCount.Add(1)
		req = clb.NewBatchRegisterTargetsRequest()
		return req, nil, errors.New("some other error")
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if callCount.Load() != 1 {
		t.Fatalf("expected 1 call (no retry for other error), got %d", callCount.Load())
	}
}

func TestIsResourceInOperatingError(t *testing.T) {
	if !IsResourceInOperatingError(mockResourceInOperatingError()) {
		t.Fatal("expected ResourceInOperating error to be detected")
	}
	if IsResourceInOperatingError(errors.New("some other error")) {
		t.Fatal("expected non-ResourceInOperating error to not be detected")
	}
}

// 验证退避间隔为指数递增（500ms, 1s, 2s, 4s, 8s）
func TestResourceInOperatingBackoffInterval(t *testing.T) {
	expected := []time.Duration{
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
	}
	for i := 1; i <= resourceInOperatingRetryMax; i++ {
		got := resourceInOperatingRetryBaseInterval * time.Duration(1<<(i-1))
		if got != expected[i-1] {
			t.Fatalf("retry %d: expected %v, got %v", i, expected[i-1], got)
		}
	}
}
