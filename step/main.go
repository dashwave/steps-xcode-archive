package step

import (
	"errors"
	"fmt"
	"os"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/dashwave/go-utils/v2/log"
)

func Run(inputs *Inputs, logFile *os.File) int {
	logger := log.NewLogger()
	logger.SetOutput(logFile)

	archiver := createXcodebuildArchiver(logger)

	config, err := archiver.ProcessInputs(*inputs)
	if err != nil {
		logger.Errorf(formattedError(fmt.Errorf("Failed to process Step inputs: %w", err)))
		return 1
	}

	dependenciesOpts := EnsureDependenciesOpts{
		XCPretty: config.LogFormatter == "xcpretty",
	}
	if err := archiver.EnsureDependencies(dependenciesOpts); err != nil {
		var xcprettyInstallErr XCPrettyInstallError
		if errors.As(err, &xcprettyInstallErr) {
			logger.Warnf("Installing xcpretty failed: %s", err)
			logger.Warnf("Switching to xcodebuild for log formatter")
			config.LogFormatter = "xcodebuild"
		} else {
			logger.Errorf(formattedError(fmt.Errorf("Failed to install Step dependencies: %w", err)))
			return 1
		}
	}

	exitCode := 0
	runOpts := createRunOptions(config)
	result, err := archiver.Run(runOpts)
	if err != nil {
		logger.Errorf(formattedError(fmt.Errorf("Failed to execute Step main logic: %w", err)))
		exitCode = 1
		// don't return as step outputs needs to be exported even in case of failure (for example the xcodebuild logs)
	}

	exportOpts := createExportOptions(config, result)
	if err := archiver.ExportOutput(exportOpts); err != nil {
		logger.Errorf(formattedError(fmt.Errorf("Failed to export Step outputs: %w", err)))
		return 1
	}

	return exitCode
}

func createXcodebuildArchiver(logger log.Logger) XcodebuildArchiver {
	xcodeVersionProvider := NewXcodebuildXcodeVersionProvider()
	envRepository := env.NewRepository()
	inputParser := stepconf.NewInputParser(envRepository)
	pathProvider := pathutil.NewPathProvider()
	pathChecker := pathutil.NewPathChecker()
	pathModifier := pathutil.NewPathModifier()
	fileManager := fileutil.NewFileManager()
	cmdFactory := command.NewFactory(envRepository)

	return NewXcodebuildArchiver(xcodeVersionProvider, inputParser, pathProvider, pathChecker, pathModifier, fileManager, logger, cmdFactory)
}

func createRunOptions(config Config) RunOpts {
	return RunOpts{
		ProjectPath:       config.ProjectPath,
		Scheme:            config.Scheme,
		Configuration:     config.Configuration,
		LogFormatter:      config.LogFormatter,
		XcodeMajorVersion: config.XcodeMajorVersion,
		ArtifactName:      config.ArtifactName,

		CodesignManager: config.CodesignManager,

		PerformCleanAction:          config.PerformCleanAction,
		XcconfigContent:             config.XcconfigContent,
		XcodebuildAdditionalOptions: config.XcodebuildAdditionalOptions,
		CacheLevel:                  config.CacheLevel,

		CustomExportOptionsPlistContent: config.ExportOptionsPlistContent,
		ExportMethod:                    config.ExportMethod,
		ICloudContainerEnvironment:      config.ICloudContainerEnvironment,
		ExportDevelopmentTeam:           config.ExportDevelopmentTeam,
		UploadBitcode:                   config.UploadBitcode,
		CompileBitcode:                  config.CompileBitcode,
	}
}

func createExportOptions(config Config, result RunResult) ExportOpts {
	return ExportOpts{
		OutputDir:      config.OutputDir,
		ArtifactName:   result.ArtifactName,
		ExportAllDsyms: config.ExportAllDsyms,

		Archive: result.Archive,

		ExportOptionsPath: result.ExportOptionsPath,
		IPAExportDir:      result.IPAExportDir,

		XcodebuildArchiveLog:       result.XcodebuildArchiveLog,
		XcodebuildExportArchiveLog: result.XcodebuildExportArchiveLog,
		IDEDistrubutionLogsDir:     result.IDEDistrubutionLogsDir,
	}
}
