package compiler

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashload/boss/consts"
	"github.com/hashload/boss/env"
	"github.com/hashload/boss/models"
	"github.com/hashload/boss/msg"
	"github.com/hashload/boss/utils"
	"github.com/hashload/boss/utils/dcp"
)

func getCompilerParameters(rootPath string, dep *models.Dependency, platform string) string {
	var binPath string
	var moduleName = ""

	if dep != nil {
		moduleName = dep.GetName()
	}

	if !env.Global {
		binPath = filepath.Join(rootPath, moduleName, consts.BinFolder)
	} else {
		binPath = env.GetGlobalBinPath()
	}

	return " /p:DCC_BplOutput=\"" + filepath.Join(rootPath, moduleName, consts.BplFolder) + "\" " +
		"/p:DCC_DcpOutput=\"" + filepath.Join(rootPath, moduleName, consts.DcpFolder) + "\" " +
		"/p:DCC_DcuOutput=\"" + filepath.Join(rootPath, moduleName, consts.DcuFolder) + "\" " +
		"/p:DCC_ExeOutput=\"" + binPath + "\" " +
		"/target:Build " +
		"/p:config=Debug " +
		"/P:platform=" + platform + " "
}

func compile(dprojPath string, dep *models.Dependency, rootLock models.PackageLock) bool {
	msg.Info("  Building " + filepath.Base(dprojPath))

	bossPackagePath := filepath.Join(env.GetModulesDir(), dep.GetName(), consts.FilePackage)

	if dependencyPackage, err := models.LoadPackageOther(bossPackagePath); err == nil {
		dcp.InjectDpcsFile(dprojPath, dependencyPackage, rootLock)
	}

	dccDir := env.GetDcc32Dir()
	rsvars := filepath.Join(dccDir, "rsvars.bat")
	fileRes := "build_boss_" + strings.TrimSuffix(filepath.Base(dprojPath), filepath.Ext(dprojPath))
	abs, _ := filepath.Abs(filepath.Dir(dprojPath))
	buildLog := filepath.Join(abs, fileRes+".log")
	buildBat := filepath.Join(abs, fileRes+".bat")
	readFile, err := ioutil.ReadFile(rsvars)
	if err != nil {
		msg.Err("    error on read rsvars.bat")
	}
	readFileStr := string(readFile)
	project, _ := filepath.Abs(dprojPath)

	readFileStr += "\n@SET DCC_UnitSearchPath=%DCC_UnitSearchPath%;" + filepath.Join(env.GetModulesDir(), consts.DcuFolder) +
		";" + filepath.Join(env.GetModulesDir(), consts.DcpFolder) //+ ";" + getNewPathsDep(dep, abs) + " "

	readFileStr += "\n@SET PATH=%PATH%;" + filepath.Join(env.GetModulesDir(), consts.BplFolder) + ";"
	for _, value := range []string{"Win32"} {
		readFileStr += " \n msbuild \"" + project + "\" /p:Configuration=Debug " + getCompilerParameters(env.GetModulesDir(), dep, value)
	}
	readFileStr += " > \"" + buildLog + "\""

	err = ioutil.WriteFile(buildBat, []byte(readFileStr), os.ModePerm)
	if err != nil {
		msg.Warn("  - error on create build file")
		return false
	}

	command := exec.Command(buildBat)
	command.Dir = abs
	if _, err := command.Output(); err != nil {
		msg.Err("  - Failed to compile, see " + buildLog + " for more information")
		return false
	} else {
		msg.Info("  - Success!")
		err := os.Remove(buildLog)
		utils.HandleError(err)
		err = os.Remove(buildBat)
		utils.HandleError(err)

		return true
	}
}
