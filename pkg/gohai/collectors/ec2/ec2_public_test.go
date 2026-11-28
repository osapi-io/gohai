// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package ec2_test

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/cloudmetadata"
	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/dmi"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/ec2"
	"github.com/osapi-io/gohai/pkg/gohai/family"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

func ec2Prior() collector.PriorResults {
	return collector.PriorResults{
		"dmi": &dmi.Info{BIOS: &dmi.BIOS{Manufacturer: "Amazon EC2"}},
	}
}

// versionListing is what GET "/" returns from EC2's IMDS. A realistic
// slice. Its newest-first order doesn't matter, because negotiation sorts it.
const versionListing = "1.0\n2011-01-01\n2019-10-01\n2021-07-15\n2099-01-01\nlatest"

// macA + macB are two canned ENIs.
const (
	macA = "02:aa:bb:cc:dd:01"
	macB = "02:aa:bb:cc:dd:02"
)

// ec2Responses is the canonical meta-data tree for the negotiated
// version "2021-07-15". Any path absent → 404.
var ec2Responses = map[string]string{
	"/2021-07-15/meta-data/ami-id":                                                      "ami-0123",
	"/2021-07-15/meta-data/ami-launch-index":                                            "0",
	"/2021-07-15/meta-data/ami-manifest-path":                                           "(unknown)",
	"/2021-07-15/meta-data/hostname":                                                    "ip-10-0-0-5.ec2.internal",
	"/2021-07-15/meta-data/instance-id":                                                 "i-abc",
	"/2021-07-15/meta-data/instance-type":                                               "t3.micro",
	"/2021-07-15/meta-data/instance-life-cycle":                                         "on-demand",
	"/2021-07-15/meta-data/local-hostname":                                              "ip-10-0-0-5.ec2.internal",
	"/2021-07-15/meta-data/local-ipv4":                                                  "10.0.0.5",
	"/2021-07-15/meta-data/mac":                                                         "0a:b0:c0:d0:e0:f0",
	"/2021-07-15/meta-data/placement/availability-zone":                                 "us-east-1a",
	"/2021-07-15/meta-data/placement/region":                                            "us-east-1",
	"/2021-07-15/meta-data/public-hostname":                                             "ec2-1-2-3-4.compute-1.amazonaws.com",
	"/2021-07-15/meta-data/public-ipv4":                                                 "1.2.3.4",
	"/2021-07-15/meta-data/reservation-id":                                              "r-abc",
	"/2021-07-15/meta-data/profile":                                                     "default-hvm",
	"/2021-07-15/meta-data/security-groups":                                             "default\nssh",
	"/2021-07-15/meta-data/local-ipv4s":                                                 "10.0.0.5\n10.0.0.6",
	"/2021-07-15/meta-data/network/interfaces/macs/":                                    macA + "/\n" + macB + "/",
	"/2021-07-15/meta-data/network/interfaces/macs/" + macA + "/device-number":          "0",
	"/2021-07-15/meta-data/network/interfaces/macs/" + macA + "/interface-id":           "eni-aaa",
	"/2021-07-15/meta-data/network/interfaces/macs/" + macA + "/local-ipv4s":            "10.0.0.5\n10.0.0.6",
	"/2021-07-15/meta-data/network/interfaces/macs/" + macA + "/subnet-id":              "subnet-111",
	"/2021-07-15/meta-data/network/interfaces/macs/" + macA + "/subnet-ipv4-cidr-block": "10.0.0.0/24",
	"/2021-07-15/meta-data/network/interfaces/macs/" + macA + "/vpc-id":                 "vpc-222",
	"/2021-07-15/meta-data/network/interfaces/macs/" + macA + "/security-groups":        "default\nssh",
	"/2021-07-15/meta-data/network/interfaces/macs/" + macB + "/interface-id":           "eni-bbb",

	// Placement extras, kernel/ramdisk, lifecycle, services, spot:
	// fields added for Ohai-parity gap closure.
	"/2021-07-15/meta-data/placement/availability-zone-id": "use1-az1",
	"/2021-07-15/meta-data/placement/group-name":           "my-cluster",
	"/2021-07-15/meta-data/placement/host-id":              "h-abc",
	"/2021-07-15/meta-data/placement/partition-number":     "1",
	"/2021-07-15/meta-data/kernel-id":                      "aki-xxxx",
	"/2021-07-15/meta-data/ramdisk-id":                     "ari-yyyy",
	"/2021-07-15/meta-data/instance-action":                "none",
	"/2021-07-15/meta-data/spot/instance-action":           "terminate",
	"/2021-07-15/meta-data/spot/termination-time":          "2026-05-01T00:00:00Z",
	"/2021-07-15/meta-data/services/domain":                "amazonaws.com",
	"/2021-07-15/meta-data/services/partition":             "aws",
	"/2021-07-15/meta-data/product-codes":                  "abcdef12345",

	// block-device-mapping tree: directory listing + per-key leaf.
	"/2021-07-15/meta-data/block-device-mapping":      "ami\nroot\nebs0",
	"/2021-07-15/meta-data/block-device-mapping/ami":  "/dev/xvda",
	"/2021-07-15/meta-data/block-device-mapping/root": "/dev/xvda1",
	"/2021-07-15/meta-data/block-device-mapping/ebs0": "/dev/xvdf",

	// public-keys tree: listing + per-index openssh-key leaf.
	"/2021-07-15/meta-data/public-keys":               "0=my-keypair",
	"/2021-07-15/meta-data/public-keys/0/openssh-key": "ssh-rsa AAAA",
}

const iamInfo = `{
  "Code": "Success",
  "LastUpdated": "2026-04-01T00:00:00Z",
  "InstanceProfileArn": "arn:aws:iam::123456789012:instance-profile/web",
  "InstanceProfileId": "AIPAXXXXXXXXXXXXXXXXX"
}`

const identityDoc = `{
  "accountId": "123456789012",
  "region": "us-east-1",
  "availabilityZone": "us-east-1a",
  "instanceId": "i-abc"
}`

type EC2PublicTestSuite struct {
	suite.Suite
}

func TestEC2PublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(EC2PublicTestSuite))
}

// hostFS is a fake host's filesystem, with /sys/hypervisor/uuid holding
// content, or no such file when content is "".
func (s *EC2PublicTestSuite) hostFS(
	content string,
) avfs.VFS {
	fs := memfs.New()
	if content != "" {
		s.Require().NoError(fs.MkdirAll("/sys/hypervisor", 0o755))
		s.Require().NoError(fs.WriteFile("/sys/hypervisor/uuid", []byte(content), 0o644))
	}
	return fs
}

// server wires a handler that implements EC2's IMDS behavior:
//   - PUT /latest/api/token → token (unless disabled)
//   - GET / → version listing
//   - GET /<ver>/meta-data/... → from responseMap (or 404)
//   - GET /<ver>/meta-data/iam/info → iamInfo
//   - GET /<ver>/dynamic/instance-identity/document → identityDoc
//   - GET /<ver>/user-data/ → userData (unless nil, 404)
type serverOpts struct {
	responseMap map[string]string // defaults to ec2Responses
	tokenBody   string            // defaults to "TOKEN-XYZ"
	token404    bool              // if true, PUT /latest/api/token returns 404 (IMDSv1 fallback)
	versionBody string            // defaults to versionListing
	version404  bool              // if true, GET / returns 404
	userData    []byte            // nil → 404
	iamMissing  bool              // if true, 404 on iam/info
	identityDoc string            // defaults to identityDoc; "" → 404
	iamBad      bool              // if true, iam/info returns bad JSON
	identityBad bool              // if true, identityDoc returns bad JSON
}

func (o *serverOpts) withDefaults() *serverOpts {
	if o.responseMap == nil {
		o.responseMap = ec2Responses
	}
	if o.tokenBody == "" && !o.token404 {
		o.tokenBody = "TOKEN-XYZ"
	}
	if o.versionBody == "" && !o.version404 {
		o.versionBody = versionListing
	}
	if o.identityDoc == "" {
		o.identityDoc = identityDoc
	}
	return o
}

func handlerFor(
	o *serverOpts,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/latest/api/token" {
			if o.token404 {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(o.tokenBody))
			return
		}
		// IMDSv2 refuses a read without the session token, as EC2 does
		// when an instance requires IMDSv2.
		if !o.token404 && r.Header.Get("X-aws-ec2-metadata-token") != o.tokenBody {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/" {
			if o.version404 {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(o.versionBody))
			return
		}
		// Route negotiated-version-specific endpoints
		for _, ver := range []string{"2021-07-15", "latest"} {
			iamPath := "/" + ver + "/meta-data/iam/info"
			idPath := "/" + ver + "/dynamic/instance-identity/document"
			udPath := "/" + ver + "/user-data/"
			if r.URL.Path == iamPath {
				if o.iamMissing {
					http.NotFound(w, r)
					return
				}
				if o.iamBad {
					_, _ = w.Write([]byte("not json"))
					return
				}
				_, _ = w.Write([]byte(iamInfo))
				return
			}
			if r.URL.Path == idPath {
				if o.identityDoc == "" {
					http.NotFound(w, r)
					return
				}
				if o.identityBad {
					_, _ = w.Write([]byte("not json"))
					return
				}
				_, _ = w.Write([]byte(o.identityDoc))
				return
			}
			if r.URL.Path == udPath {
				if o.userData == nil {
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write(o.userData)
				return
			}
		}
		if body, ok := o.responseMap[r.URL.Path]; ok {
			_, _ = w.Write([]byte(body))
			return
		}
		http.NotFound(w, r)
	}
}

func (s *EC2PublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		validateFunc func(ec2.Collector)
	}{
		{
			name: "New gives the ec2 cloud collector, opt-in, after dmi",
			validateFunc: func(c ec2.Collector) {
				s.Equal("ec2", c.Name())
				s.Equal(collector.CategoryCloud, c.Category())
				s.False(c.DefaultEnabled())
				s.Equal([]string{"dmi"}, c.Dependencies())
				s.NotNil(c.(*ec2.Metadata).FS)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(ec2.New())
		})
	}
}

func (s *EC2PublicTestSuite) TestCollect() {
	huge := strings.Repeat("x", cloudmetadata.MaxBody+1)
	withBody := func(path, body string) map[string]string {
		m := maps.Clone(ec2Responses)
		m["/2021-07-15"+path] = body

		return m
	}

	// unsafeMap lists names that would change the request path or query.
	unsafeMap := maps.Clone(ec2Responses)
	unsafeMap["/2021-07-15/meta-data/block-device-mapping"] = "ami\n../../user-data\nroot?x=1"
	unsafeMap["/2021-07-15/meta-data/public-keys"] = "0=my-keypair\n..=bad\n1#frag=bad"
	unsafeMap["/2021-07-15/meta-data/network/interfaces/macs/"] = macA + "/\n../\nx y/"
	var unsafeMu sync.Mutex
	var unsafeSeen []string
	unsafeHandler := func(w http.ResponseWriter, r *http.Request) {
		unsafeMu.Lock()
		unsafeSeen = append(
			unsafeSeen,
			r.URL.RawPath+r.URL.Path+"?"+r.URL.RawQuery+"#"+r.URL.Fragment,
		)
		unsafeMu.Unlock()
		handlerFor((&serverOpts{responseMap: unsafeMap}).withDefaults())(w, r)
	}

	tests := []struct {
		name         string
		prior        collector.PriorResults
		hypervisor   string // content for /sys/hypervisor/uuid; "" = don't write
		family       string // the host's family; "" is Linux
		opts         *serverOpts
		handler      http.HandlerFunc // overrides opts-based handler when set
		closed       bool
		cancelAfter  int           // cancel the context once the server answered this many requests
		timeout      time.Duration // the collector's walk timeout; zero is WalkTimeout
		wantErr      string        // the error Collect returns; "" is none. With validateFunc, the fact is partial
		validateFunc func(any, bool)
	}{
		{
			name: "a walk that outlives its timeout is an error",
			// The server never answers, so only the walk's deadline can end
			// the collection.
			handler: func(_ http.ResponseWriter, r *http.Request) {
				<-r.Context().Done()
			},
			timeout: time.Millisecond,
			wantErr: "context deadline exceeded",
		},
		{
			name:        "a context that ends during the walk is an error",
			opts:        (&serverOpts{}).withDefaults(),
			cancelAfter: 5,
			wantErr:     "context canceled",
		},
		{
			name:    "a leaf that is more than MaxBody keeps the metadata read before it",
			opts:    (&serverOpts{responseMap: withBody("/meta-data/security-groups", huge)}).withDefaults(),
			wantErr: "read ec2 metadata /meta-data/security-groups: ",
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.NotEmpty(info.ID)
				s.Nil(info.SecurityGroups)
			},
		},
		{
			name:    "a first path that is more than MaxBody is an error",
			opts:    (&serverOpts{responseMap: withBody("/meta-data/ami-id", huge)}).withDefaults(),
			wantErr: "read ec2 metadata /meta-data/ami-id: ",
		},
		{
			name: "IMDSv2 happy path with version negotiation + full walk",
			opts: (&serverOpts{
				userData: []byte("plain user-data"),
			}).withDefaults(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal("2021-07-15", info.APIVersion)
				s.Equal("i-abc", info.ID)
				s.Equal("t3.micro", info.Type)
				s.Equal("10.0.0.5", info.LocalIPv4)
				s.Equal([]string{"10.0.0.5", "10.0.0.6"}, info.LocalIPv4s)
				s.Equal([]string{"default", "ssh"}, info.SecurityGroups)
				s.Equal("123456789012", info.AccountUID)
				s.Equal("us-east-1", info.Region)
				s.Equal("us-east-1a", info.Zone)
				s.Equal("use1-az1", info.AvailabilityZoneID)
				s.Equal("my-cluster", info.GroupName)
				s.Equal("h-abc", info.HostID)
				s.Equal("1", info.PartitionNumber)
				s.Equal("aki-xxxx", info.KernelID)
				s.Equal("ari-yyyy", info.RamdiskID)
				s.Equal("none", info.InstanceAction)
				s.Equal("terminate", info.SpotInstanceAction)
				s.Equal("2026-05-01T00:00:00Z", info.SpotTerminationTime)
				s.Equal("amazonaws.com", info.ServicesDomain)
				s.Equal("aws", info.CloudPartition)
				s.Equal([]string{"abcdef12345"}, info.ProductCodes)
				s.Equal("/dev/xvda", info.BlockDeviceMapping["ami"])
				s.Equal("/dev/xvda1", info.BlockDeviceMapping["root"])
				s.Equal("/dev/xvdf", info.BlockDeviceMapping["ebs0"])
				s.Equal([]string{"ssh-rsa AAAA"}, info.PublicKeys)
				s.Require().NotNil(info.IAMInfo)
				s.Equal(
					"arn:aws:iam::123456789012:instance-profile/web",
					info.IAMInfo.InstanceProfileArn,
				)
				// The user-data is a secret: the facts leave it out.
				facts, err := json.Marshal(info)
				s.Require().NoError(err)
				s.NotContains(string(facts), "plain user-data")

				s.Require().Len(info.NetworkInterfaces, 2)
				a, okA := info.NetworkInterfaces[macA]
				s.Require().True(okA)
				s.Equal("eni-aaa", a.InterfaceID)
				s.Equal("0", a.DeviceNumber)
				s.Equal([]string{"10.0.0.5", "10.0.0.6"}, a.LocalIPv4s)
				s.Equal("subnet-111", a.SubnetID)
				s.Equal("vpc-222", a.VPCID)
				b, okB := info.NetworkInterfaces[macB]
				s.Require().True(okB)
				s.Equal("eni-bbb", b.InterfaceID)
			},
		},
		{
			name: "IMDSv1 fallback when token PUT 404s",
			opts: (&serverOpts{token404: true}).withDefaults(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal("i-abc", info.ID)
			},
		},
		{
			name: "version negotiation falls back to 'latest' on 404",
			opts: (&serverOpts{
				version404: true,
				responseMap: func() map[string]string {
					// Produce an equivalent tree under /latest/
					m := make(map[string]string, len(ec2Responses))
					for k, v := range ec2Responses {
						nk := "/latest" + k[len("/2021-07-15"):]
						m[nk] = v
					}
					return m
				}(),
			}).withDefaults(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal("latest", info.APIVersion)
			},
		},
		{
			name: "version negotiation falls back to 'latest' when no intersection",
			opts: (&serverOpts{
				versionBody: "2099-01-01\n2099-02-01",
				responseMap: func() map[string]string {
					m := make(map[string]string, len(ec2Responses))
					for k, v := range ec2Responses {
						nk := "/latest" + k[len("/2021-07-15"):]
						m[nk] = v
					}
					return m
				}(),
			}).withDefaults(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal("latest", info.APIVersion)
			},
		},
		{
			name:    "a listed name that would change the request path is skipped",
			handler: unsafeHandler,
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal(map[string]string{"ami": "/dev/xvda"}, info.BlockDeviceMapping)
				s.Equal([]string{"ssh-rsa AAAA"}, info.PublicKeys)
				s.Len(info.NetworkInterfaces, 1)
				unsafeMu.Lock()
				defer unsafeMu.Unlock()
				for _, seen := range unsafeSeen {
					s.NotContains(seen, "..")
					s.NotContains(seen, "x y")
					s.True(strings.HasSuffix(seen, "?#"))
				}
			},
		},
		{
			name: "block-device-mapping listing with all-404 children yields nil",
			opts: (&serverOpts{
				responseMap: func() map[string]string {
					m := make(map[string]string, len(ec2Responses))
					for k, v := range ec2Responses {
						if strings.HasPrefix(k, "/2021-07-15/meta-data/block-device-mapping/") {
							continue // drop per-key leaves so all fetches 404
						}
						m[k] = v
					}
					// Override listing to only contain unknown entries.
					m["/2021-07-15/meta-data/block-device-mapping"] = "ghost0\nghost1"
					return m
				}(),
			}).withDefaults(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Nil(info.BlockDeviceMapping)
			},
		},
		{
			name: "public-keys listing without = delimiter + per-key 404",
			opts: (&serverOpts{
				responseMap: func() map[string]string {
					m := make(map[string]string, len(ec2Responses))
					for k, v := range ec2Responses {
						m[k] = v
					}
					// First line has no `=` (idx = line), second line's leaf 404s.
					m["/2021-07-15/meta-data/public-keys"] = "0\n1=phantom"
					m["/2021-07-15/meta-data/public-keys/0/openssh-key"] = "ssh-rsa BARE"
					// index "1" has no leaf in the map → 404
					return m
				}(),
			}).withDefaults(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal([]string{"ssh-rsa BARE"}, info.PublicKeys)
			},
		},
		{
			name: "dmi says not EC2 and no xen UUID short-circuits",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{BIOS: &dmi.BIOS{Manufacturer: "Dell Inc."}},
			},
			opts: (&serverOpts{}).withDefaults(),
			validateFunc: func(out any, httpCalled bool) {
				s.False(httpCalled)
				s.Exactly(nil, out)
			},
		},
		{
			name: "detection via bios_version substring",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{BIOS: &dmi.BIOS{Manufacturer: "Xen", Ver: "4.2.amazon"}},
			},
			opts: (&serverOpts{}).withDefaults(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal("i-abc", info.ID)
			},
		},
		{
			name: "detection via hypervisor UUID prefix",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{BIOS: &dmi.BIOS{Manufacturer: "Xen"}},
			},
			hypervisor: "ec2-abc-def",
			opts:       (&serverOpts{}).withDefaults(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal("i-abc", info.ID)
			},
		},
		{
			name: "hypervisor UUID without ec2 prefix doesn't match",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{BIOS: &dmi.BIOS{Manufacturer: "Xen"}},
			},
			hypervisor: "kvm-12345",
			opts:       (&serverOpts{}).withDefaults(),
			validateFunc: func(out any, httpCalled bool) {
				s.False(httpCalled)
				s.Exactly(nil, out)
			},
		},
		{
			name:   "macOS, where dmi has no tables, probes the metadata service",
			prior:  collector.PriorResults{"dmi": &dmi.Info{}},
			family: family.Darwin,
			opts:   (&serverOpts{}).withDefaults(),
			validateFunc: func(out any, httpCalled bool) {
				s.True(httpCalled)
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Equal("i-abc", info.ID)
			},
		},
		{
			name:  "Linux with an empty dmi and no xen UUID is not EC2",
			prior: collector.PriorResults{"dmi": &dmi.Info{}},
			opts:  (&serverOpts{}).withDefaults(),
			validateFunc: func(out any, httpCalled bool) {
				s.False(httpCalled)
				s.Exactly(nil, out)
			},
		},
		{
			name: "no BIOS in dmi still triggers hypervisor UUID check",
			prior: collector.PriorResults{
				"dmi": &dmi.Info{},
			},
			hypervisor: "ec2-abc",
			opts:       (&serverOpts{}).withDefaults(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Equal("i-abc", info.ID)
			},
		},
		{
			name:  "no dmi in prior fails open",
			prior: collector.PriorResults{},
			opts:  (&serverOpts{}).withDefaults(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Equal("i-abc", info.ID)
			},
		},
		{
			name: "first-path 404 drops silently",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPut {
					_, _ = w.Write([]byte("T"))
					return
				}
				if r.URL.Path == "/" {
					_, _ = w.Write([]byte("latest"))
					return
				}
				http.NotFound(w, r)
			},
			validateFunc: func(out any, _ bool) {
				s.Exactly(nil, out)
			},
		},
		{
			name:    "connection refused drops silently",
			handler: func(http.ResponseWriter, *http.Request) {},
			closed:  true,
			validateFunc: func(out any, _ bool) {
				s.Exactly(nil, out)
			},
		},
		{
			name: "iam info missing tolerated",
			opts: (&serverOpts{iamMissing: true}).withDefaults(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Nil(info.IAMInfo)
			},
		},
		{
			name: "malformed iam JSON tolerated",
			opts: (&serverOpts{iamBad: true}).withDefaults(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Nil(info.IAMInfo)
			},
		},
		{
			name: "identity doc missing tolerated",
			opts: func() *serverOpts {
				o := (&serverOpts{}).withDefaults()
				o.identityDoc = ""
				return o
			}(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				// meta-data/placement/region still populates it
				s.Equal("us-east-1", info.Region)
				s.Empty(info.AccountUID)
			},
		},
		{
			name: "malformed identity doc JSON tolerated",
			opts: (&serverOpts{identityBad: true}).withDefaults(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Equal("us-east-1", info.Region)
				s.Empty(info.AccountUID)
			},
		},
		{
			name: "identity doc fills missing fields",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Only ami-id + identity-doc + token + listing respond
				if r.Method == http.MethodPut {
					_, _ = w.Write([]byte("T"))
					return
				}
				if r.URL.Path == "/" {
					_, _ = w.Write([]byte(versionListing))
					return
				}
				if r.URL.Path == "/2021-07-15/meta-data/ami-id" {
					_, _ = w.Write([]byte("ami-0123"))
					return
				}
				if r.URL.Path == "/2021-07-15/dynamic/instance-identity/document" {
					_, _ = w.Write([]byte(identityDoc))
					return
				}
				http.NotFound(w, r)
			},
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Equal("i-abc", info.ID)
				s.Equal("us-east-1", info.Region)
				s.Equal("us-east-1a", info.Zone)
				s.Equal("123456789012", info.AccountUID)
			},
		},
		{
			name: "macs tree missing is tolerated",
			opts: func() *serverOpts {
				o := (&serverOpts{}).withDefaults()
				// Strip the macs/ listing so fetchENIs returns early
				m := make(map[string]string, len(o.responseMap))
				for k, v := range o.responseMap {
					if k == "/2021-07-15/meta-data/network/interfaces/macs/" {
						continue
					}
					m[k] = v
				}
				o.responseMap = m
				return o
			}(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Empty(info.NetworkInterfaces)
			},
		},
		{
			name: "security-groups scrub key is dropped",
			opts: func() *serverOpts {
				o := (&serverOpts{}).withDefaults()
				m := make(map[string]string, len(o.responseMap))
				for k, v := range o.responseMap {
					m[k] = v
				}
				m["/2021-07-15/meta-data/security-groups"] = "default\nidentity_credentials_ec2_security_credentials_ec2_instance\nssh"
				o.responseMap = m
				return o
			}(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Equal([]string{"default", "ssh"}, info.SecurityGroups)
			},
		},
		{
			name: "local-ipv4s missing is tolerated",
			opts: func() *serverOpts {
				o := (&serverOpts{}).withDefaults()
				m := make(map[string]string, len(o.responseMap))
				for k, v := range o.responseMap {
					if k == "/2021-07-15/meta-data/local-ipv4s" {
						continue
					}
					m[k] = v
				}
				o.responseMap = m
				return o
			}(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Nil(info.LocalIPv4s)
			},
		},
		{
			name: "security-groups missing is tolerated",
			opts: func() *serverOpts {
				o := (&serverOpts{}).withDefaults()
				m := make(map[string]string, len(o.responseMap))
				for k, v := range o.responseMap {
					if k == "/2021-07-15/meta-data/security-groups" {
						continue
					}
					m[k] = v
				}
				o.responseMap = m
				return o
			}(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Nil(info.SecurityGroups)
			},
		},
		{
			name: "version listing returns empty body falls back to latest",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPut {
					_, _ = w.Write([]byte("T"))
					return
				}
				if r.URL.Path == "/" {
					// An empty 200: splitLines returns nil → fallback.
					_, _ = w.Write([]byte(""))
					return
				}
				if r.URL.Path == "/latest/meta-data/ami-id" {
					_, _ = w.Write([]byte("ami-fallback"))
					return
				}
				http.NotFound(w, r)
			},
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal("latest", info.APIVersion)
				s.Equal("ami-fallback", info.ImageID)
			},
		},
		{
			name: "security-groups containing only the scrub key produces nil",
			opts: func() *serverOpts {
				o := (&serverOpts{}).withDefaults()
				m := make(map[string]string, len(o.responseMap))
				for k, v := range o.responseMap {
					m[k] = v
				}
				m["/2021-07-15/meta-data/security-groups"] = "identity_credentials_ec2_security_credentials_ec2_instance"
				o.responseMap = m
				return o
			}(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Nil(info.SecurityGroups)
			},
		},
		{
			name: "empty listing lines in macs tree are skipped",
			opts: func() *serverOpts {
				o := (&serverOpts{}).withDefaults()
				m := make(map[string]string, len(o.responseMap))
				for k, v := range o.responseMap {
					m[k] = v
				}
				m["/2021-07-15/meta-data/network/interfaces/macs/"] = "\n" + macA + "/\n\n"
				o.responseMap = m
				return o
			}(),
			validateFunc: func(out any, _ bool) {
				info, ok := out.(*ec2.Info)
				s.Require().True(ok)
				s.Len(info.NetworkInterfaces, 1)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// The handler runs on the server's goroutines, so the flag
			// that it sets is atomic.
			var httpCalled atomic.Bool
			h := tt.handler
			if h == nil {
				h = handlerFor(tt.opts)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					httpCalled.Store(true)
					h(w, r)
					if tt.cancelAfter > 0 && int(requests.Add(1)) == tt.cancelAfter {
						cancel()
					}
				},
			))
			if tt.closed {
				srv.Close()
			} else {
				defer srv.Close()
			}

			c := ec2.NewWith(source.Sources{
				HTTP:   sourcetest.Server(srv.URL),
				FS:     s.hostFS(tt.hypervisor),
				Family: tt.family,
			})

			prior := tt.prior
			if prior == nil {
				prior = ec2Prior()
			}
			if tt.timeout > 0 {
				c.(*ec2.Metadata).Timeout = tt.timeout
			}
			out, err := c.Collect(ctx, prior)
			if tt.wantErr != "" {
				s.ErrorContains(err, tt.wantErr)
				if tt.validateFunc == nil {
					s.Exactly(nil, out)

					return
				}
				tt.validateFunc(out, httpCalled.Load())

				return
			}
			s.Require().NoError(err)

			tt.validateFunc(out, httpCalled.Load())
		})
	}
}
