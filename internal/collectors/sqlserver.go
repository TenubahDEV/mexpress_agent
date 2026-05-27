package collectors

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"sync"

	_ "github.com/microsoft/go-mssqldb"
)

type SQLServerMetrics struct {
	Up                float64
	UserConnections   float64
	DatabaseCount     float64
	FailedJobsLast24h float64
	Databases         []SQLDatabaseMetrics
}

type SQLDatabaseMetrics struct {
	Name                    string
	Status                  float64 // 1 = ONLINE, 0 = NOT ONLINE
	SizeMB                  float64
	LogSizeMB               float64
	LogUsedPercent          float64
	LastFullBackupAgeHours  float64
	LastDiffBackupAgeHours  float64
	LastLogBackupAgeMinutes float64
}

var (
	dbInstance *sql.DB
	dbMu       sync.Mutex
)

// getSQLConnection retorna una conexión pool reutilizable a SQL Server (Thread-Safe)
func getSQLConnection(connStr string) (*sql.DB, error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	if dbInstance != nil {
		return dbInstance, nil
	}

	// Ajustar la cadena de conexión para Windows si se usa autenticación de Windows
	// Go-mssqldb soporta Trusted_Connection=yes o Integrated Security=true en Windows
	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return nil, err
	}

	// Límites del pool seguros y ligeros
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // mantendremos vivas las conexiones ociosas

	dbInstance = db
	return db, nil
}

// CollectSQLServer realiza las consultas de monitoreo de SQL Server de forma segura y compatible
func CollectSQLServer(ctx context.Context, connStr string) (*SQLServerMetrics, error) {
	m := &SQLServerMetrics{
		Up: 0,
	}

	db, err := getSQLConnection(connStr)
	if err != nil {
		log.Printf("ERROR sqlserver: Failed to open SQL Server connection: %v", err)
		return m, err
	}

	// 1. Ping para validar si el motor está arriba
	if err := db.PingContext(ctx); err != nil {
		log.Printf("ERROR sqlserver: Failed to ping SQL Server: %v", err)
		return m, err // Retorna Up = 0
	}

	m.Up = 1

	// 2. Obtener conexiones de usuario activas
	err = db.QueryRowContext(ctx, "SELECT COUNT(session_id) FROM sys.dm_exec_sessions WHERE is_user_process = 1").Scan(&m.UserConnections)
	if err != nil {
		log.Printf("ERROR sqlserver: Failed to query user connections: %v", err)
	}

	// 3. Obtener cantidad total de bases de datos
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sys.databases").Scan(&m.DatabaseCount)
	if err != nil {
		log.Printf("ERROR sqlserver: Failed to query database count: %v", err)
	}

	// 4. Obtener trabajos fallidos en las últimas 24h
	failedJobsQuery := `
		SELECT COALESCE(COUNT(DISTINCT sj.job_id), 0)
		FROM msdb.dbo.sysjobs sj
		JOIN msdb.dbo.sysjobhistory sjh ON sj.job_id = sjh.job_id
		WHERE sjh.run_status = 0
		  AND sjh.step_id = 0
		  AND msdb.dbo.agent_datetime(sjh.run_date, sjh.run_time) >= DATEADD(day, -1, GETDATE())`
	err = db.QueryRowContext(ctx, failedJobsQuery).Scan(&m.FailedJobsLast24h)
	if err != nil {
		// En Express editions o si el Agent está apagado, esta query fallará. Lo registramos como advertencia.
		log.Printf("WARNING sqlserver: Failed to query Agent Jobs history (SQL Server Agent might be disabled): %v", err)
		m.FailedJobsLast24h = 0
	}

	// 5. Monitoreo por Base de Datos
	dbMap := make(map[string]*SQLDatabaseMetrics)

	// Consulta A: Estados y Tamaños (Data y Log)
	sizeQuery := `
		SELECT 
			d.name,
			CASE WHEN d.state = 0 THEN 1 ELSE 0 END,
			SUM(CASE WHEN f.type = 0 THEN f.size * 8 / 1024 ELSE 0 END),
			SUM(CASE WHEN f.type = 1 THEN f.size * 8 / 1024 ELSE 0 END)
		FROM sys.databases d
		LEFT JOIN sys.master_files f ON d.database_id = f.database_id
		GROUP BY d.name, d.state`

	rows, err := db.QueryContext(ctx, sizeQuery)
	if err != nil {
		log.Printf("ERROR sqlserver: Failed to query database sizes: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var dbName string
			var status, sizeMB, logSizeMB float64
			if err := rows.Scan(&dbName, &status, &sizeMB, &logSizeMB); err == nil {
				dbName = strings.TrimSpace(dbName)
				dbMap[dbName] = &SQLDatabaseMetrics{
					Name:                    dbName,
					Status:                  status,
					SizeMB:                  sizeMB,
					LogSizeMB:               logSizeMB,
					LastFullBackupAgeHours:  -1,
					LastDiffBackupAgeHours:  -1,
					LastLogBackupAgeMinutes: -1,
				}
			}
		}
	}

	// Consulta B: Porcentaje de uso del Log (Compatibilidad Universal por Performance Counters)
	logUsedQuery := `
		SELECT 
			RTRIM(instance_name),
			cntr_value
		FROM sys.dm_os_performance_counters
		WHERE counter_name = 'Percent Log Used'
		  AND instance_name <> '_Total'`

	rowsLog, err := db.QueryContext(ctx, logUsedQuery)
	if err != nil {
		log.Printf("WARNING sqlserver: Failed to query log usage performance counters: %v", err)
	} else {
		defer rowsLog.Close()
		for rowsLog.Next() {
			var dbName string
			var logUsed float64
			if err := rowsLog.Scan(&dbName, &logUsed); err == nil {
				dbName = strings.TrimSpace(dbName)
				if dbMetric, exists := dbMap[dbName]; exists {
					dbMetric.LogUsedPercent = logUsed
				}
			}
		}
	}

	// Consulta C: Edades de Backups
	backupQuery := `
		SELECT 
			d.name,
			COALESCE(DATEDIFF(hour, MAX(CASE WHEN b.type = 'D' THEN b.backup_finish_date END), GETDATE()), -1),
			COALESCE(DATEDIFF(hour, MAX(CASE WHEN b.type = 'I' THEN b.backup_finish_date END), GETDATE()), -1),
			COALESCE(DATEDIFF(minute, MAX(CASE WHEN b.type = 'L' THEN b.backup_finish_date END), GETDATE()), -1)
		FROM sys.databases d
		LEFT JOIN msdb.dbo.backupset b ON d.name = b.database_name
		GROUP BY d.name`

	rowsBackup, err := db.QueryContext(ctx, backupQuery)
	if err != nil {
		log.Printf("WARNING sqlserver: Failed to query database backup age: %v", err)
	} else {
		defer rowsBackup.Close()
		for rowsBackup.Next() {
			var dbName string
			var fullAge, diffAge, logAge float64
			if err := rowsBackup.Scan(&dbName, &fullAge, &diffAge, &logAge); err == nil {
				dbName = strings.TrimSpace(dbName)
				if dbMetric, exists := dbMap[dbName]; exists {
					dbMetric.LastFullBackupAgeHours = fullAge
					dbMetric.LastDiffBackupAgeHours = diffAge
					dbMetric.LastLogBackupAgeMinutes = logAge
				}
			}
		}
	}

	// Convertir el mapa en un slice final
	for _, dbMetric := range dbMap {
		m.Databases = append(m.Databases, *dbMetric)
	}

	// En Windows, si cerramos la conexión se pueden generar fugas en hilos de background del pool.
	// El recolector mantendrá abierta la conexión del pool persistente.
	return m, nil
}

// Close SQL connections when agent terminates (best effort)
func Close() {
	dbMu.Lock()
	defer dbMu.Unlock()
	if dbInstance != nil {
		dbInstance.Close()
		dbInstance = nil
	}
}
