package server

import (
	"main/utils"
	"strings"

	"github.com/robfig/cron/v3"
)

func (w *workerInstance) RunCronJob() {
	c := cron.New()

	// Create cronjob
	workers := utils.AppConfig.Workers
	for _, worker := range workers {
		if worker.Enable {
			cronName := strings.TrimSpace(worker.Name)
			cronFunc := w.mappingWorkers()[worker.Function]
			cronTime := strings.TrimSpace(worker.TimeRun)
			c.AddFunc(cronTime, cronFunc)
			utils.ShowInfoLogs("Init worker [%s], time [%s] successful", cronName, cronTime)
		}
	}

	c.Start()
}

func (w *workerInstance) mappingWorkers() map[string]func() {
	mapHandler := make(map[string]func())
	mapHandler["createEcommerceOrder"] = w.importSvc.ImportExcel
	mapHandler["scanPlatformOrder"] = w.platformSvc.CronjobScanOrder
	mapHandler["scanPlatformOrderPending"] = w.platformSvc.CronjobScanOrderPending
	mapHandler["scanPlatformOrderRefund"] = w.platformSvc.CronjobScanOrderRefund
	mapHandler["scanPlatformOrderToShip"] = w.platformSvc.CronjobScanOrderToShip
	mapHandler["scanPlatformOrderPayment"] = w.platformSvc.CronjobScanOrderPayment
	mapHandler["scanStatementTransaction"] = w.platformSvc.CronjobScanStatementTransaction

	mapHandler["checkExpiredTokenLazada"] = w.lazadaSvc.CheckExpiredTokenLazada
	mapHandler["checkExpiredTokenTiktok"] = w.tiktokSvc.CheckExpiredTokenTiktok

	mapHandler["cleanupAuthSessions"] = w.authSvc.CleanupAuthSessions
	return mapHandler
}
