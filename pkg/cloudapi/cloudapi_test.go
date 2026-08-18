package cloudapi

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

func setEnvironment(t *testing.T, value string) {
	t.Helper()
	if err := SetEnvironment(value); err != nil {
		t.Fatalf("set environment %q failed: %v", value, err)
	}
}

func TestNewClientProfile(t *testing.T) {
	t.Run("默认现网内网域名", func(t *testing.T) {
		setEnvironment(t, EnvironmentProd)
		cases := map[string]string{
			"clb": "clb.internal.tencentcloudapi.com",
			"vpc": "vpc.internal.tencentcloudapi.com",
			"cam": "cam.internal.tencentcloudapi.com",
			"tag": "tag.internal.tencentcloudapi.com",
		}
		for svc, want := range cases {
			p := NewClientProfile(svc)
			if p.HttpProfile.Endpoint != want {
				t.Errorf("service %s: expect endpoint %q, got %q", svc, want, p.HttpProfile.Endpoint)
			}
		}
	})

	t.Run("测试环境内网域名", func(t *testing.T) {
		setEnvironment(t, EnvironmentTest)
		defer setEnvironment(t, EnvironmentProd)
		cases := map[string]string{
			"clb": "clb.internal.test.tencentcloudapi.com",
			"vpc": "vpc.internal.test.tencentcloudapi.com",
			"cam": "cam.internal.test.tencentcloudapi.com",
			"tag": "tag.internal.test.tencentcloudapi.com",
		}
		for svc, want := range cases {
			p := NewClientProfile(svc)
			if p.HttpProfile.Endpoint != want {
				t.Errorf("service %s: expect endpoint %q, got %q", svc, want, p.HttpProfile.Endpoint)
			}
		}
	})

	t.Run("测试环境 TLS SNI 兼容证书", func(t *testing.T) {
		setEnvironment(t, EnvironmentTest)
		defer setEnvironment(t, EnvironmentProd)
		cases := map[string]string{
			"clb": "clb.test.tencentcloudapi.com",
			"vpc": "vpc.test.tencentcloudapi.com",
			"cam": "cam.test.tencentcloudapi.com",
			"tag": "tag.test.tencentcloudapi.com",
		}
		for svc, want := range cases {
			transport := NewHTTPTransport(svc)
			if transport.TLSClientConfig == nil || transport.TLSClientConfig.ServerName != want {
				t.Errorf("service %s: expect TLS SNI %q, got %#v", svc, want, transport.TLSClientConfig)
			}
		}
	})

	t.Run("现网不覆盖 TLS SNI", func(t *testing.T) {
		setEnvironment(t, EnvironmentProd)
		transport := NewHTTPTransport("clb")
		if transport.TLSClientConfig != nil && transport.TLSClientConfig.ServerName != "" {
			t.Fatalf("expect empty TLS SNI, got %q", transport.TLSClientConfig.ServerName)
		}
	})

	t.Run("非法环境", func(t *testing.T) {
		setEnvironment(t, EnvironmentTest)
		defer setEnvironment(t, EnvironmentProd)
		for _, value := range []string{"", "invalid", "PROD", "TEST"} {
			if err := SetEnvironment(value); err == nil {
				t.Errorf("set environment %q: expect an error", value)
			}
		}
		if environment != EnvironmentTest {
			t.Fatalf("invalid environment must not change current environment, got %q", environment)
		}
	})
}

// TestHTTPTransportKeepsTestHostAndSNI 验证测试模式下 HTTP Host 与 TLS 校验名分离且证书校验仍开启。
func TestHTTPTransportKeepsTestHostAndSNI(t *testing.T) {
	const (
		httpHost = "clb.internal.test.tencentcloudapi.com"
		tlsName  = "clb.test.tencentcloudapi.com"
	)

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate private key failed: %v", err)
	}
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("generate serial number failed: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: tlsName},
		DNSNames:     []string{tlsName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("create certificate failed: %v", err)
	}
	certificate, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("parse certificate failed: %v", err)
	}
	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(certificate)

	type tlsRequest struct {
		host string
		sni  string
	}
	gotRequest := make(chan tlsRequest, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sni := ""
		if r.TLS != nil {
			sni = r.TLS.ServerName
		}
		gotRequest <- tlsRequest{host: r.Host, sni: sni}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Response":{"QuotaSet":[],"RequestId":"tls-sni-test"}}`))
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{{
		Certificate: [][]byte{certDER},
		PrivateKey:  privateKey,
	}}}
	server.StartTLS()
	defer server.Close()

	setEnvironment(t, EnvironmentTest)
	defer setEnvironment(t, EnvironmentProd)
	profile := NewClientProfile("clb")
	if profile.HttpProfile.Endpoint != httpHost {
		t.Fatalf("expect endpoint %q, got %q", httpHost, profile.HttpProfile.Endpoint)
	}
	client, err := clb.NewClient(common.NewCredential("testSecretId", "testSecretKey"), "ap-guangzhou", profile)
	if err != nil {
		t.Fatalf("new clb client failed: %v", err)
	}
	transport := NewHTTPTransport("clb")
	transport.TLSClientConfig.RootCAs = rootCAs
	serverAddress := strings.TrimPrefix(server.URL, "https://")
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, serverAddress)
	}
	client.WithHttpTransport(transport)

	if _, err := client.DescribeQuota(clb.NewDescribeQuotaRequest()); err != nil {
		t.Fatalf("describe quota failed: %v", err)
	}
	request := <-gotRequest
	if request.host != httpHost {
		t.Errorf("expect HTTP Host %q, got %q", httpHost, request.host)
	}
	if request.sni != tlsName {
		t.Errorf("expect TLS SNI %q, got %q", tlsName, request.sni)
	}
	if transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("certificate verification must not be disabled")
	}
}

// TestNewClientProfileEndpointReached 验证设置 Endpoint 后 SDK 请求确实打到该域名（集成行为）
func TestNewClientProfileEndpointReached(t *testing.T) {
	gotHost := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost <- r.Host
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Response":{"TotalCount":0,"RequestId":"test-request-id"}}`))
	}))
	defer srv.Close()

	setEnvironment(t, EnvironmentTest)
	defer setEnvironment(t, EnvironmentProd)
	p := NewClientProfile("clb")
	p.HttpProfile.Scheme = "HTTP"
	p.HttpProfile.Endpoint = strings.TrimPrefix(srv.URL, "http://")

	client, err := clb.NewClient(common.NewCredential("testSecretId", "testSecretKey"), "ap-guangzhou", p)
	if err != nil {
		t.Fatalf("new clb client failed: %v", err)
	}
	if _, err := client.DescribeLoadBalancers(clb.NewDescribeLoadBalancersRequest()); err != nil {
		t.Fatalf("describe load balancers failed: %v", err)
	}
	if host := <-gotHost; host != strings.TrimPrefix(srv.URL, "http://") {
		t.Errorf("expect request sent to %q, got %q", strings.TrimPrefix(srv.URL, "http://"), host)
	}
}
