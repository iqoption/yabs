package api

import (
	"github.com/gin-gonic/gin"
	"fmt"
	"net/http"
	"io/ioutil"
	"io"
	"os"
	"time"
	"yabs/collector/cfg"
	"yabs/collector/service"
	"encoding/json"
	"bytes"
	log "github.com/sirupsen/logrus"
	"github.com/iqoption/ginmm"
)

type BaseReply struct {
	Status string `json:"status"`
}

type GinCollectorService struct {
	engine  *gin.Engine
	conf    cfg.Config
	service *service.CollectorService
}

type SymbolDescription  struct {
	Version string `json:"version"`
	Platform string `json:"platform"`
}

type UploadParams struct {
	context *gin.Context
	param   string
	prefix  string
	tmpDir  string
}

func (m *GinCollectorService) Init() error {
	cfg.GlobalConfigMutex.Lock()
	defer cfg.GlobalConfigMutex.Unlock()

	var err error = nil
	m.conf = cfg.GlobalConfig
	m.engine = gin.Default()
	mm := ginmm.NewMetricMiddleware(ginmm.MetricParams{
		Service:         "crashes",
		UdpAddres:       m.conf.UdpAddress(),
		FlushBufferSize: m.conf.FlushBufferSize(),
		FlushTimeout:    time.Duration(m.conf.FlushTimeout()),
	})

	m.engine.Use(mm.Middleware())

	m.service, err = service.NewCollector(m.conf)
	if err != nil {
		return err
	}
	os.MkdirAll(m.conf.DumpsTmpDir(), 0777)
	os.MkdirAll(m.conf.SymbolsTmpDir(), 0777)

	m.applyRoutes()
	return nil
}

func (m *GinCollectorService) setSuccessStatus(c *gin.Context) {
	rMsg := &BaseReply{"success"}
	c.JSON(http.StatusOK, rMsg)
}
func (m *GinCollectorService) setServerError(descr string, c *gin.Context) {
	rMsg := &BaseReply{fmt.Sprintf("error: %s", descr)}
	c.JSON(http.StatusInternalServerError, rMsg)
}

func (m *GinCollectorService) setBadRequest(descr string, c *gin.Context) {
	rMsg := &BaseReply{fmt.Sprintf("error: %s", descr)}
	c.JSON(http.StatusBadRequest, rMsg)
}

func (m *GinCollectorService) Start() error {
	addres := fmt.Sprintf("%s:%d", m.conf.Host(), m.conf.Port())
	log.WithField("address", addres).Info("Run on")
	return m.engine.Run(addres)
}

func (m *GinCollectorService) applyRoutes() {
	m.engine.POST("/symbols", m.PostSymbol())
	m.engine.POST("/submit", m.PostMiniDump())
	m.engine.POST("/submit/web", m.PostWebDump())
}

func (m *GinCollectorService) PostSymbol() gin.HandlerFunc {
	return func(c *gin.Context) {

		description, _, err := c.Request.FormFile("description")
		if err != nil {
			m.setBadRequest("Missing parameter 'description'", c)
			return
		}
		defer description.Close()

		tpmDescript, err := ioutil.TempFile(m.conf.SymbolsTmpDir(), m.prefix("description_"))
		if err != nil {
			log.WithError(err).Error("Could not create temporary file")
			m.setServerError("Could not create temporary file", c)
			return
		}

		defer tpmDescript.Close()

		var buf bytes.Buffer
		w := io.MultiWriter(tpmDescript, &buf)

		_, err = io.Copy(w, description)
		if err != nil {
			defer os.Remove(tpmDescript.Name())

			log.WithError(err).Error("Could not write to temporary description file")
			m.setServerError("Could not create temporary file", c)
			return
		}

		descr := m.readDescrition(&buf)
		if descr == nil {
			defer os.Remove(tpmDescript.Name())

			m.setBadRequest("Description invalid format", c)
			return
		}

		symbolPath, err := m.uploadFile(UploadParams{
			context: c,
			param:   "file",
			prefix:  m.prefix("symbol_"),
			tmpDir:  m.conf.SymbolsTmpDir(),
		})

		if err != nil {
			defer os.Remove(tpmDescript.Name())
			m.setBadRequest("Can't upload 'file'", c)
			return
		}

		wasmSymbolPath := ""
		if descr.Platform == "web" {
			wasmSymbolPath, _ = m.uploadFile(UploadParams{
				context: c,
				param:   "file2",
				prefix:  m.prefix("symbol_"),
				tmpDir:  m.conf.SymbolsTmpDir(),
			})
		}

		if len(wasmSymbolPath) == 0 {
			err = m.service.AddSymbol(symbolPath, tpmDescript.Name())
		} else {
			symPaths := []string{
				symbolPath,
				wasmSymbolPath,
			}

			err = m.service.AddSymbols(symPaths, tpmDescript.Name())
		}

		if err != nil {
			defer os.Remove(symbolPath)
			defer os.Remove(tpmDescript.Name())

			if len(wasmSymbolPath) != 0 {
				defer os.Remove(wasmSymbolPath)
			}

			m.setServerError("Can't add new task to process symbol files", c)
		} else {
			log.WithFields(log.Fields{
				"platform":    descr.Platform,
				"version":     descr.Version,
				"symbolfile":  symbolPath,
				"descritpion": tpmDescript.Name(),
			}).Debug("Send symbol to processor")
			m.setSuccessStatus(c)
		}
	}
}

func (m *GinCollectorService) PostMiniDump() gin.HandlerFunc {
	return func(c *gin.Context) {

		dumpPath, err := m.uploadFile(UploadParams{
			context: c,
			param:   "upload_file_minidump",
			prefix:  m.prefix("minidump_"),
			tmpDir:  m.conf.DumpsTmpDir(),
		})

		if err != nil {
			m.setBadRequest("Can't upload 'upload_file_minidump'", c)
			return
		}

		infoPath, err := m.uploadFile(UploadParams{
			context: c,
			param:   "info",
			prefix:  m.prefix("info_"),
			tmpDir:  m.conf.DumpsTmpDir(),
		})

		if err != nil {
			m.setBadRequest("Can't upload 'info'", c)
			defer os.Remove(dumpPath)
			return
		}

		logPath, err := m.uploadFile(UploadParams{
			context: c,
			param:   "log",
			prefix:  m.prefix("log_"),
			tmpDir:  m.conf.DumpsTmpDir(),
		})

		err = m.service.AddMinidump(dumpPath, infoPath, logPath)
		if err != nil {
			defer os.Remove(dumpPath)
			defer os.Remove(infoPath)

			m.setServerError("Can't add new task to process minidump files", c)
		} else {
			m.setSuccessStatus(c)
		}
	}
}

func (m *GinCollectorService) PostWebDump() gin.HandlerFunc {
	return func(c *gin.Context) {
		stack := c.Request.FormValue("uploaded")
		if stack == "" {
			c.String(http.StatusBadRequest, "Field uploaded can't be empty")
			return
		}

		infoData := c.Request.FormValue("info")
		if infoData == "" {
			c.String(http.StatusBadRequest, "Field info can't be empty")
			return
		}
		log.WithFields(log.Fields{
			"stack": stack,
			"info":  infoData,
		}).Debug("Catch web dump")

		var info map[string]interface{}
		err := json.Unmarshal([]byte(infoData), &info)
		if err != nil {
			log.WithField("web-info", infoData).Debug("Invalid info fromat. Need json")
			c.String(http.StatusBadRequest, "Invalid info fromat. Need json")
			return
		}

		tmpDump, err := ioutil.TempFile(m.conf.DumpsTmpDir(), m.prefix("webdump_"))
		if err != nil {
			log.WithError(err).Error("Could not create temporary web-dump file")
			c.String(http.StatusInternalServerError, "Could not create temporary file")
			return
		}

		_, err = tmpDump.WriteString(stack)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  tmpDump.Name(),
			}).Error("Can't write web-dump to file")
			m.closeAndRemove(tmpDump)
			return
		}

		tmpInfo, err := ioutil.TempFile(m.conf.DumpsTmpDir(), m.prefix("webinfo_"))
		if err != nil {
			m.closeAndRemove(tmpDump)
			log.WithError(err).Error("Could not create temporary info file")
			c.String(http.StatusInternalServerError, "Could not create temporary file")
			return
		}

		newData, err := json.Marshal(&info)
		if err != nil {
			m.closeAndRemove(tmpDump)
			m.closeAndRemove(tmpInfo)

			log.WithError(err).Error("Could not serialize info")
			c.String(http.StatusInternalServerError, "Could not serialize info")
			return
		}

		_, err = tmpInfo.Write(newData)
		if err != nil {
			m.closeAndRemove(tmpDump)
			m.closeAndRemove(tmpInfo)

			log.WithError(err).Error("Could not create temporary info file")
			c.String(http.StatusInternalServerError, "Could not create temporary file")
			return
		}

		tmpDump.Close()
		tmpInfo.Close()
		err = m.service.AddWebDump(tmpDump.Name(), tmpInfo.Name())
		if err != nil {
			m.closeAndRemove(tmpDump)
			m.closeAndRemove(tmpInfo)

			m.setServerError("Can't add new task to process minidump files", c)
		} else {
			m.setSuccessStatus(c)
		}
	}
}

func (m *GinCollectorService) closeAndRemove(f *os.File) {
	err := f.Close()
	if err == nil {
		os.Remove(f.Name())
	} else {
		log.WithFields(log.Fields{
			"error":     err,
			"file name": f.Name(),
		}).Error("Can't close file")
	}
}

func (m *GinCollectorService) prefix(p string) string {
	dt := time.Now()
	return fmt.Sprintf("%04d%02d%02d%02d%02d-%s", dt.Year(),
		dt.Month(),
		dt.Day(),
		dt.Hour(),
		dt.Minute(),
		p)
}

func (m *GinCollectorService) readDescrition(buff *bytes.Buffer) *SymbolDescription {
	d := &SymbolDescription{
	}

	err := json.Unmarshal(buff.Bytes(), d)
	if err != nil {
		log.WithError(err).Error("Can't parse symbols descripton")
		return nil
	}

	return d
}

func (m *GinCollectorService) uploadFile(args UploadParams) (string, error) {
	file, _, err := args.context.Request.FormFile(args.param)
	if err != nil {
		log.WithField("param", args.param).
			Warning("Upload file: missing parameter")
		return "", err
	}
	defer file.Close()

	tmpFile, err := ioutil.TempFile(args.tmpDir,
		m.prefix(args.prefix))

	if err != nil {
		log.WithError(err).Error("Could not create temporary file")
		return "", err
	}

	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, file)
	if err != nil {
		defer os.Remove(tmpFile.Name())
		log.WithFields(log.Fields{}).
			Error("Could not create temporary file")
		return "", err
	}

	tmpFile.Close()
	return tmpFile.Name(), nil
}