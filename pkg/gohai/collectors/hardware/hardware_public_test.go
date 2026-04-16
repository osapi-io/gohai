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

package hardware_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/hardware"
)

var (
	_ collector.Collector = (*hardware.Linux)(nil)
	_ collector.Collector = (*hardware.Darwin)(nil)
)

const hwJSONAppleSilicon = `{
  "SPHardwareDataType": [{
    "_name": "hardware_overview",
    "machine_model": "MacBookPro18,2",
    "machine_name": "MacBook Pro",
    "serial_number": "F5K123ABC",
    "platform_UUID": "00000000-1111-2222-3333-444444444444",
    "provisioning_UDID": "12345",
    "chip_type": "Apple M1 Pro",
    "number_processors": "proc 10:8:2",
    "physical_memory": "32 GB",
    "boot_rom_version": "10151.101.3",
    "os_loader_version": "10151.101.3"
  }]
}`

const hwJSONIntel = `{
  "SPHardwareDataType": [{
    "_name": "hardware_overview",
    "machine_model": "MacBookPro11,1",
    "cpu_type": "Intel Core i7",
    "current_processor_speed": "3 GHz",
    "number_processors": 2,
    "packages": 1,
    "l2_cache_core": "256 KB",
    "l3_cache": "4 MB",
    "physical_memory": "16 GB",
    "serial_number": "ABC123",
    "SMC_version_system": "2.16f68",
    "boot_rom_version": "MBP111.0138.B17"
  }]
}`

const storageJSON = `{
  "SPStorageDataType": [
    {
      "_name": "Macintosh HD",
      "bsd_name": "disk1s1",
      "size_in_bytes": 249661751296,
      "free_space_in_bytes": 123456789012,
      "file_system": "APFS",
      "mount_point": "/",
      "writable": "yes"
    },
    {
      "_name": "Legacy HD",
      "bsd_name": "disk0s2",
      "size_in_bytes": 500000000000,
      "writable": "yes",
      "com.apple.corestorage.pv": [
        {"medium_type": "ssd", "smart_status": "Verified"},
        {"medium_type": "ssd", "smart_status": "Verified"}
      ]
    }
  ]
}`

const powerJSON = `{
  "SPPowerDataType": [
    {
      "_name": "spbattery_information",
      "sppower_current_amperage": 0,
      "sppower_current_voltage": 12788,
      "sppower_battery_charge_info": {
        "sppower_battery_current_capacity": 5000,
        "sppower_battery_max_capacity": 6000,
        "sppower_battery_fully_charged": "FALSE",
        "sppower_battery_is_charging": "TRUE"
      },
      "sppower_battery_health_info": {
        "sppower_battery_cycle_count": 201,
        "sppower_battery_health": "Good"
      },
      "sppower_battery_model_info": {
        "sppower_battery_serial_number": "D123456789ABCDEFG"
      }
    },
    {
      "_name": "sppower_ac_charger_information",
      "sppower_ac_charger_ID": "0x0100",
      "sppower_ac_charger_family": "0x0085",
      "sppower_ac_charger_revision": "0x0000",
      "sppower_ac_charger_serial_number": "0x00a1dab7",
      "sppower_ac_charger_watts": "85",
      "sppower_battery_charger_connected": "TRUE"
    },
    {
      "_name": "sppower_hwconfig_information",
      "sppower_ups_installed": "FALSE"
    }
  ]
}`

// hwExec returns a MockExecutor keyed by the data-type arg so the
// same mock answers all three system_profiler calls in one test case.
func hwExec(
	t *testing.T,
	hwOut, storageOut, powerOut []byte,
	hwErr, storageErr, powerErr error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "system_profiler", "SPHardwareDataType", "-json").
		Return(hwOut, hwErr).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "system_profiler", "SPStorageDataType", "-json").
		Return(storageOut, storageErr).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), "system_profiler", "SPPowerDataType", "-json").
		Return(powerOut, powerErr).
		AnyTimes()
	return m
}

type HardwarePublicTestSuite struct {
	suite.Suite
}

func TestHardwarePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HardwarePublicTestSuite))
}

func (s *HardwarePublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string
	}{
		{"darwin dispatches to Darwin", "darwin", "darwin"},
		{"debian dispatches to Linux", "debian", "linux"},
		{"unknown dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := hardware.New()
			s.Equal("hardware", c.Name())
			s.Equal("hardware", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*hardware.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*hardware.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *HardwarePublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		variant  string
		exec     func(*testing.T) executor.Executor
		validate func(*hardware.Info)
	}{
		{
			name:    "darwin: happy path Apple Silicon + APFS + battery + charger",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return hwExec(
					t,
					[]byte(hwJSONAppleSilicon),
					[]byte(storageJSON),
					[]byte(powerJSON),
					nil,
					nil,
					nil,
				)
			},
			validate: func(info *hardware.Info) {
				s.Equal("MacBookPro18,2", info.MachineModel)
				s.Equal("Apple M1 Pro", info.ChipType)
				s.Equal("F5K123ABC", info.SerialNumber)
				s.Equal("00000000-1111-2222-3333-444444444444", info.PlatformUUID)
				s.Equal("proc 10:8:2", info.NumberProcessors)
				s.Equal("32 GB", info.PhysicalMemory)

				s.Require().Len(info.Storage, 2)
				s.Equal("Macintosh HD", info.Storage[0].Name)
				s.Equal(int64(249661751296), info.Storage[0].Capacity)
				s.Equal("APFS", info.Storage[0].FileSystem)
				s.True(info.Storage[0].Writable)
				s.Empty(info.Storage[0].DriveType) // APFS, no CoreStorage

				s.Equal("Legacy HD", info.Storage[1].Name)
				s.Equal("ssd", info.Storage[1].DriveType)
				s.Equal("Verified", info.Storage[1].SmartStatus)
				s.Equal(2, info.Storage[1].Partitions)

				s.Require().NotNil(info.Battery)
				s.Equal(5000, info.Battery.CurrentCapacity)
				s.Equal(6000, info.Battery.MaxCapacity)
				s.False(info.Battery.FullyCharged)
				s.True(info.Battery.IsCharging)
				s.Equal(201, info.Battery.ChargeCycleCount)
				s.Equal("Good", info.Battery.Health)
				s.Equal("D123456789ABCDEFG", info.Battery.Serial)
				s.Equal(83, info.Battery.Remaining) // 5000 * 100 / 6000
				s.Equal(12788, info.Battery.Voltage)

				s.Require().NotNil(info.Charger)
				s.Equal("0x0100", info.Charger.ID)
				s.Equal("85", info.Charger.Watts)
				s.True(info.Charger.Connected)
			},
		},
		{
			name:    "darwin: Intel Mac populates cpu_type/packages",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return hwExec(t, []byte(hwJSONIntel), []byte(`{}`), []byte(`{}`), nil, nil, nil)
			},
			validate: func(info *hardware.Info) {
				s.Equal("Intel Core i7", info.CPUType)
				s.Equal("3 GHz", info.CurrentProcessorSpeed)
				s.Equal("2", info.NumberProcessors)
				s.Equal(1, info.Packages)
				s.Equal("2.16f68", info.SMCVersionSystem)
				s.Equal("MBP111.0138.B17", info.BootROMVersion)
				s.Nil(info.Battery)
				s.Nil(info.Charger)
			},
		},
		{
			name:    "darwin: all three calls fail → empty Info",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return hwExec(t, nil, nil, nil,
					errors.New("no"), errors.New("no"), errors.New("no"))
			},
			validate: func(info *hardware.Info) {
				s.Empty(info.MachineModel)
				s.Empty(info.Storage)
				s.Nil(info.Battery)
				s.Nil(info.Charger)
			},
		},
		{
			name:    "darwin: malformed hardware JSON tolerated",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return hwExec(t, []byte("not json"), []byte(`{}`), []byte(`{}`), nil, nil, nil)
			},
			validate: func(info *hardware.Info) {
				s.Empty(info.MachineModel)
			},
		},
		{
			name:    "darwin: empty SPHardwareDataType items array tolerated",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return hwExec(
					t,
					[]byte(`{"SPHardwareDataType": []}`),
					[]byte(`{}`),
					[]byte(`{}`),
					nil,
					nil,
					nil,
				)
			},
			validate: func(info *hardware.Info) {
				s.Empty(info.MachineModel)
			},
		},
		{
			name:    "darwin: storage without CoreStorage (modern APFS only)",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return hwExec(
					t,
					[]byte(`{}`),
					[]byte(
						`{"SPStorageDataType": [{"_name": "Data", "bsd_name": "disk2", "size_in_bytes": 1000}]}`,
					),
					[]byte(`{}`),
					nil,
					nil,
					nil,
				)
			},
			validate: func(info *hardware.Info) {
				s.Require().Len(info.Storage, 1)
				s.Empty(info.Storage[0].DriveType)
				s.Empty(info.Storage[0].SmartStatus)
				s.Equal(0, info.Storage[0].Partitions)
			},
		},
		{
			name:    "darwin: malformed storage JSON tolerated",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return hwExec(t, []byte(`{}`), []byte("not json"), []byte(`{}`), nil, nil, nil)
			},
			validate: func(info *hardware.Info) {
				s.Empty(info.Storage)
			},
		},
		{
			name:    "darwin: malformed power JSON tolerated",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return hwExec(t, []byte(`{}`), []byte(`{}`), []byte("not json"), nil, nil, nil)
			},
			validate: func(info *hardware.Info) {
				s.Nil(info.Battery)
				s.Nil(info.Charger)
			},
		},
		{
			name:    "darwin: max_capacity zero → remaining stays 0",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return hwExec(t, []byte(`{}`), []byte(`{}`), []byte(`{
                    "SPPowerDataType": [{
                        "_name": "spbattery_information",
                        "sppower_battery_charge_info": {
                            "sppower_battery_current_capacity": 0,
                            "sppower_battery_max_capacity": 0
                        }
                    }]
                }`), nil, nil, nil)
			},
			validate: func(info *hardware.Info) {
				s.Require().NotNil(info.Battery)
				s.Equal(0, info.Battery.Remaining)
			},
		},
		{
			name:    "darwin: nil Exec yields empty",
			variant: "darwin",
			exec:    func(*testing.T) executor.Executor { return nil },
			validate: func(info *hardware.Info) {
				s.Empty(info.MachineModel)
			},
		},
		{
			name:    "darwin: int-as-string fields parse correctly",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return hwExec(
					t,
					[]byte(`{"SPHardwareDataType": [{"packages": "4"}]}`),
					[]byte(`{}`),
					[]byte(`{}`),
					nil,
					nil,
					nil,
				)
			},
			validate: func(info *hardware.Info) {
				s.Equal(4, info.Packages)
			},
		},
		{
			name:    "darwin: battery ints and booleans in odd JSON shapes",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				// cycle_count as string, amperage missing entirely,
				// fully_charged as native bool, is_charging as "yes".
				return hwExec(t, []byte(`{}`), []byte(`{}`), []byte(`{
                    "SPPowerDataType": [{
                        "_name": "spbattery_information",
                        "sppower_battery_charge_info": {
                            "sppower_battery_current_capacity": 80,
                            "sppower_battery_max_capacity": 100,
                            "sppower_battery_fully_charged": true,
                            "sppower_battery_is_charging": "yes"
                        },
                        "sppower_battery_health_info": {
                            "sppower_battery_cycle_count": "42"
                        }
                    }]
                }`), nil, nil, nil)
			},
			validate: func(info *hardware.Info) {
				s.Require().NotNil(info.Battery)
				s.Equal(42, info.Battery.ChargeCycleCount)
				s.True(info.Battery.FullyCharged)
				s.True(info.Battery.IsCharging)
			},
		},
		{
			name:    "darwin: battery unparseable string cycle count leaves zero",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return hwExec(t, []byte(`{}`), []byte(`{}`), []byte(`{
                    "SPPowerDataType": [{
                        "_name": "spbattery_information",
                        "sppower_battery_health_info": {
                            "sppower_battery_cycle_count": "not-a-number"
                        }
                    }]
                }`), nil, nil, nil)
			},
			validate: func(info *hardware.Info) {
				s.Require().NotNil(info.Battery)
				s.Equal(0, info.Battery.ChargeCycleCount)
			},
		},
		{
			name:    "darwin: number_processors as float parses to string",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return hwExec(
					t,
					[]byte(`{"SPHardwareDataType": [{"number_processors": 8}]}`),
					[]byte(`{}`),
					[]byte(`{}`),
					nil,
					nil,
					nil,
				)
			},
			validate: func(info *hardware.Info) {
				s.Equal("8", info.NumberProcessors)
			},
		},
		{
			name:    "linux: returns empty",
			variant: "linux",
			validate: func(info *hardware.Info) {
				s.Require().NotNil(info)
				s.Empty(info.MachineModel)
				s.Empty(info.Storage)
				s.Nil(info.Battery)
				s.Nil(info.Charger)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c hardware.Collector
			switch tt.variant {
			case "darwin":
				c = &hardware.Darwin{Exec: tt.exec(s.T())}
			case "linux":
				c = &hardware.Linux{}
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			info, ok := got.(*hardware.Info)
			s.Require().True(ok)
			tt.validate(info)
		})
	}
}
