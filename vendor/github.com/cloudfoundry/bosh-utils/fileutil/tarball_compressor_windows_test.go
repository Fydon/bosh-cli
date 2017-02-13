// +build windows

package fileutil_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

func fixtureSrcTgzTarCompatible() string {
	pwd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	return "/" + strings.Join([]string{strings.Replace(strings.Replace(pwd, "\\", "/", -1), ":", "", 1), "test_assets", "compressor-decompress-file-to-dir.tgz"}, "/")
}

var _ = Describe("tarballCompressor", func() {
	var (
		dstDir       string
		dstDirForTar string
		cmdRunner    boshsys.CmdRunner
		fs           boshsys.FileSystem
		compressor   Compressor
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		cmdRunner = boshsys.NewExecCmdRunner(logger)
		fs = boshsys.NewOsFileSystem(logger)
		tmpDir, err := fs.TempDir("tarballCompressor-test")
		Expect(err).NotTo(HaveOccurred())
		dstDir = filepath.Join(tmpDir, "TestCompressor")
		dstDirForTar = "/" + strings.Join([]string{strings.Replace(strings.Replace(tmpDir, "\\", "/", -1), ":", "", 1), "TestCompressor"}, "/")
		compressor = NewTarballCompressor(cmdRunner, fs)
	})

	BeforeEach(func() {
		fs.MkdirAll(dstDir, os.ModePerm)
	})

	AfterEach(func() {
		fs.RemoveAll(dstDir)
	})

	Describe("CompressFilesInDir", func() {
		It("compresses the files in the given directory", func() {
			srcDir := fixtureSrcDir()

			symlinkPath, err := createTestSymlink()
			Expect(err).To(Succeed())
			defer os.Remove(symlinkPath)

			tgzName, err := compressor.CompressFilesInDir(srcDir)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tgzName)

			tgzNameForTar := strings.Replace("/"+string(tgzName[0])+string(tgzName[2:]), "\\", "/", -1)

			tarballContents, _, _, err := cmdRunner.RunCommand("tar", "-tf", tgzNameForTar)
			Expect(err).ToNot(HaveOccurred())

			contentElements := strings.Fields(strings.TrimSpace(tarballContents))

			Expect(contentElements).To(ConsistOf(
				"./",
				"./app.stderr.log",
				"./app.stdout.log",
				"./other_logs/",
				"./some_directory/",
				"./some_directory/sub_dir/",
				"./some_directory/sub_dir/other_sub_dir/",
				"./some_directory/sub_dir/other_sub_dir/.keep",
				"./symlink_dir",
				"./other_logs/more_logs/",
				"./other_logs/other_app.stderr.log",
				"./other_logs/other_app.stdout.log",
				"./other_logs/more_logs/more.stdout.log",
			))

			_, _, _, err = cmdRunner.RunCommand("tar", "-xzpf", tgzNameForTar, "-C", dstDirForTar)
			Expect(err).ToNot(HaveOccurred())

			content, err := fs.ReadFileString(dstDir + "/app.stdout.log")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stdout"))

			content, err = fs.ReadFileString(dstDir + "/app.stderr.log")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stderr"))

			content, err = fs.ReadFileString(dstDir + "/other_logs/other_app.stdout.log")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is other app stdout"))
		})
	})

	Describe("CompressSpecificFilesInDir", func() {
		It("compresses the given files in the given directory", func() {
			srcDir := fixtureSrcDir()
			files := []string{
				"app.stdout.log",
				"some_directory",
				"app.stderr.log",
			}
			tgzName, err := compressor.CompressSpecificFilesInDir(srcDir, files)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tgzName)

			tgzNameForTar := strings.Replace("/"+string(tgzName[0])+string(tgzName[2:]), "\\", "/", -1)

			tarballContents, _, _, err := cmdRunner.RunCommand("tar", "-tf", tgzNameForTar)
			Expect(err).ToNot(HaveOccurred())

			contentElements := strings.Fields(strings.TrimSpace(tarballContents))

			Expect(contentElements).To(Equal([]string{
				"app.stdout.log",
				"some_directory/",
				"some_directory/sub_dir/",
				"some_directory/sub_dir/other_sub_dir/",
				"some_directory/sub_dir/other_sub_dir/.keep",
				"app.stderr.log",
			}))

			_, _, _, err = cmdRunner.RunCommand("cp", tgzName, "/tmp")

			_, _, _, err = cmdRunner.RunCommand("tar", "-xzpf", tgzNameForTar, "-C", dstDirForTar)
			Expect(err).ToNot(HaveOccurred())

			content, err := fs.ReadFileString(dstDir + "/app.stdout.log")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stdout"))

			content, err = fs.ReadFileString(dstDir + "/app.stderr.log")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is app stderr"))

			content, err = fs.ReadFileString(dstDir + "/some_directory/sub_dir/other_sub_dir/.keep")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("this is a .keep file"))
		})
	})

	Describe("DecompressFileToDir", func() {
		It("decompresses the file to the given directory", func() {
			err := compressor.DecompressFileToDir(fixtureSrcTgz(), dstDir, CompressorOptions{})
			Expect(err).ToNot(HaveOccurred())

			content, err := fs.ReadFileString(dstDir + "/not-nested-file")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("not-nested-file"))

			content, err = fs.ReadFileString(dstDir + "/dir/nested-file")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("nested-file"))

			content, err = fs.ReadFileString(dstDir + "/dir/nested-dir/double-nested-file")
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(ContainSubstring("double-nested-file"))

			Expect(dstDir + "/empty-dir").To(beDir())
			Expect(dstDir + "/dir/empty-nested-dir").To(beDir())
		})

		It("returns error if the destination does not exist", func() {
			fs.RemoveAll(dstDir)

			err := compressor.DecompressFileToDir(fixtureSrcTgz(), dstDir, CompressorOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(dstDirForTar))
		})

		It("uses no same owner option", func() {
			cmdRunner := fakesys.NewFakeCmdRunner()
			compressor := NewTarballCompressor(cmdRunner, fs)

			tarballPath := fixtureSrcTgz()
			err := compressor.DecompressFileToDir(tarballPath, dstDir, CompressorOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(1).To(Equal(len(cmdRunner.RunCommands)))
			Expect(cmdRunner.RunCommands[0]).To(Equal(
				[]string{
					"tar", "--no-same-owner",
					"-xzvf", fixtureSrcTgzTarCompatible(),
					"-C", dstDirForTar,
				},
			))
		})

		It("uses same owner option", func() {
			cmdRunner := fakesys.NewFakeCmdRunner()
			compressor := NewTarballCompressor(cmdRunner, fs)

			tarballPath := fixtureSrcTgz()
			err := compressor.DecompressFileToDir(
				tarballPath,
				dstDir,
				CompressorOptions{SameOwner: true},
			)
			Expect(err).ToNot(HaveOccurred())

			Expect(1).To(Equal(len(cmdRunner.RunCommands)))
			Expect(cmdRunner.RunCommands[0]).To(Equal(
				[]string{
					"tar", "--same-owner",
					"-xzvf", fixtureSrcTgzTarCompatible(),
					"-C", dstDirForTar,
				},
			))
		})
	})

	Describe("CleanUp", func() {
		It("removes tarball path", func() {
			fs := fakesys.NewFakeFileSystem()
			compressor := NewTarballCompressor(cmdRunner, fs)

			err := fs.WriteFileString("/fake-tarball.tar", "")
			Expect(err).ToNot(HaveOccurred())

			err = compressor.CleanUp("/fake-tarball.tar")
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists("/fake-tarball.tar")).To(BeFalse())
		})

		It("returns error if removing tarball path fails", func() {
			fs := fakesys.NewFakeFileSystem()
			compressor := NewTarballCompressor(cmdRunner, fs)

			fs.RemoveAllStub = func(_ string) error {
				return errors.New("fake-remove-all-err")
			}

			err := compressor.CleanUp("/fake-tarball.tar")
			Expect(err).To(MatchError("fake-remove-all-err"))
		})
	})
})
