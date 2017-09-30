package cloud_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"

	. "github.com/cloudfoundry/bosh-cli/cloud"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CpiCmdRunner", func() {
	var (
		cpiCmdRunner CPICmdRunner
		context      CmdContext
		cmdRunner    *fakesys.FakeCmdRunner
		cpi          CPI
	)

	BeforeEach(func() {
		context = CmdContext{
			DirectorID: "fake-director-id",
		}

		cpi = CPI{
			JobPath:     filepath.Join("/", "jobs", "cpi"),
			JobsDir:     filepath.Join("/", "jobs"),
			PackagesDir: filepath.Join("/", "packages"),
		}

		cmdRunner = fakesys.NewFakeCmdRunner()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		cpiCmdRunner = NewCPICmdRunner(cmdRunner, cpi, logger)
	})

	Describe("Run", func() {
		It("creates correct command", func() {
			cmdOutput := CmdOutput{}
			outputBytes, err := json.Marshal(cmdOutput)
			Expect(err).NotTo(HaveOccurred())

			result := fakesys.FakeCmdResult{
				Stdout:     string(outputBytes),
				ExitStatus: 0,
			}
			cmdRunner.AddCmdResult(filepath.Join("/", "jobs", "cpi", "bin", "cpi"), result)

			_, err = cpiCmdRunner.Run(context, "fake-method", "fake-argument-1", "fake-argument-2")
			Expect(err).NotTo(HaveOccurred())
			Expect(cmdRunner.RunComplexCommands).To(HaveLen(1))

			actualCmd := cmdRunner.RunComplexCommands[0]
			if runtime.GOOS == "windows" {
				workingDir, _ := os.Getwd()
				workingDir = strings.Replace(workingDir, ":", "", -1)
				decodedRune, n := utf8.DecodeRuneInString(workingDir)
				workingDir = "/" + filepath.ToSlash(string(unicode.ToLower(decodedRune)) + workingDir[n:]) + "/"
				Expect(actualCmd.Name).To(Equal(filepath.Join("/", "jobs", "cpi", "bin", "cpi")))
				Expect(actualCmd.Args).To([]string{
					"-c",
					"\"export BOSH_PACKAGES_DIR='" + workingDir + filepath.ToSlash(cpi.PackagesDir) + "';" +
					" export BOSH_JOBS_DIR='" + workingDir + filepath.ToSlash(cpi.JobsDir) + "';" +
					" export PATH='/usr/local/bin:/usr/bin:/bin:/sbin:/c/Windows/System32/WindowsPowerShell/v1.0:/d/Program Files/Oracle/VirtualBox'; bash -x /jobs/cpi/bin/cpi\"",
				})
				Expect(actualCmd.Env).To(BeNil())
				Expect(actualCmd.UseIsolatedEnv).To(BeFalse())
			} else {
				Expect(actualCmd.Name).To(Equal("/jobs/cpi/bin/cpi")))
				Expect(actualCmd.Args).To(BeNil())
				Expect(actualCmd.Env).To(Equal(map[string]string{
					"BOSH_PACKAGES_DIR": cpi.PackagesDir,
					"BOSH_JOBS_DIR":     cpi.JobsDir,
					"PATH":              "/usr/local/bin:/usr/bin:/bin:/sbin",
				}))
				Expect(actualCmd.UseIsolatedEnv).To(BeTrue())
			}
			bytes, err := ioutil.ReadAll(actualCmd.Stdin)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(bytes)).To(Equal(
				`{` +
					`"method":"fake-method",` +
					`"arguments":["fake-argument-1","fake-argument-2"],` +
					`"context":{"director_uuid":"fake-director-id"}` +
					`}`,
			))
		})

		Context("when the command succeeds", func() {
			BeforeEach(func() {
				cmdOutput := CmdOutput{
					Result: "fake-cid",
				}
				outputBytes, err := json.Marshal(cmdOutput)
				Expect(err).NotTo(HaveOccurred())

				result := fakesys.FakeCmdResult{
					Stdout:     string(outputBytes),
					ExitStatus: 0,
				}
				cmdRunner.AddCmdResult(filepath.Join("/", "jobs", "cpi", "bin", "cpi"), result)
			})

			It("returns the result", func() {
				cmdOutput, err := cpiCmdRunner.Run(context, "fake-method", "fake-argument")
				Expect(err).NotTo(HaveOccurred())
				Expect(cmdOutput).To(Equal(CmdOutput{
					Result: "fake-cid",
					Error:  nil,
					Log:    "",
				}))
			})
		})

		Context("when running the command fails", func() {
			BeforeEach(func() {
				result := fakesys.FakeCmdResult{
					Error: errors.New("fake-error-trying-to-run-command"),
				}
				cmdRunner.AddCmdResult(filepath.Join("/", "jobs", "cpi", "bin", "cpi"), result)
			})

			It("returns an error", func() {
				_, err := cpiCmdRunner.Run(context, "fake-method", "fake-argument")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-error-trying-to-run-command"))
			})
		})

		Context("when the command runs but fails", func() {
			BeforeEach(func() {
				cmdOutput := CmdOutput{
					Error: &CmdError{
						Message: "fake-run-error",
					},
					Result: "fake-cid",
				}
				outputBytes, err := json.Marshal(cmdOutput)
				Expect(err).NotTo(HaveOccurred())

				result := fakesys.FakeCmdResult{
					Stdout:     string(outputBytes),
					ExitStatus: 0,
				}
				cmdRunner.AddCmdResult(filepath.Join("/", "jobs", "cpi", "bin", "cpi"), result)
			})

			It("returns the command output and no error", func() {
				cmdOutput, err := cpiCmdRunner.Run(context, "fake-method", "fake-argument")
				Expect(err).ToNot(HaveOccurred())
				Expect(cmdOutput.Error.Message).To(ContainSubstring("fake-run-error"))
			})
		})
	})
})
