// Copyright (C) 2018 Cranky Kernel
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package pkg

import (
	"fmt"
	"database/sql"
	"time"
	"encoding/json"
	"github.com/crankykernel/maker/pkg/log"
)

var db *sql.DB

func incrementVersion(tx *sql.Tx, version int) error {
	_, err := tx.Exec("insert into schema values (?, 'now')", version)
	return err
}

func initDb(db *sql.DB) error {
	var version = 0
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	row := tx.QueryRow("select max(version) from schema")
	if err := row.Scan(&version); err != nil {
		log.Printf("Initializing database.")
		_, err := db.Exec("create table schema (version integer not null primary key, timestamp timestamp)")
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create schema table: %v", err)
		}
		if err := incrementVersion(tx, 0); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert into schema table: %v", err)
		}
		version = 0
	} else {
		log.Printf("Found database version %d.", version)
	}

	if version < 1 {
		_, err := tx.Exec(`create table binance_raw_execution_report (timestamp timestamp, report json);`)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create binance_raw_execution_reports table: %v", err)
		}
		_, err = tx.Exec(`create index binance_raw_execution_report_timestamp_index on binance_raw_execution_report(timestamp)`)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create binance_raw_execution_reports_timestamp_index: %v", err)
		}
		if err := incrementVersion(tx, 1); err != nil {
			tx.Rollback()
			return err
		}
	}

	if version < 2 {
		_, err := tx.Exec(`create table binance_trade (id string primary key unique, archived bool default false, data json)`)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create binance_trade table: %v", err)
		}
		if err := incrementVersion(tx, 2); err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()

	return nil
}

func DbOpen() {
	var err error
	db, err = sql.Open("sqlite3", "maker.db")
	if err != nil {
		log.Fatal(err)
	}
	if err := initDb(db); err != nil {
		log.Fatal(err)
	}
}

func DbSaveBinanceRawExecutionReport(event *UserStreamEvent) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`insert into binance_raw_execution_report (timestamp, report) values (?, ?)`,
		formatTimestamp(event.EventTime), event.Raw)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func DbSaveTrade(trade *Trade) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	data, err := formatJson(trade.State)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`insert into binance_trade (id, data) values (?, ?)`,
		trade.State.LocalID, data)
	tx.Commit()
	return err
}

func DbUpdateTrade(trade *Trade) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	data, err := formatJson(trade.State)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`update binance_trade set data = ? where id = ?`,
		data, trade.State.LocalID)
	if err != nil {
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func DbArchiveTrade(trade *Trade) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`update binance_trade set archived = 1 where id = ?`,
		trade.State.LocalID)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return err
}

func DbRestoreTradeState() ([]TradeState, error) {
	rows, err := db.Query(`select id, data from binance_trade where archived = 0`)
	if err != nil {
		return nil, err
	}

	tradeStates := []TradeState{}

	for rows.Next() {
		var localId string
		var data string
		if err := rows.Scan(&localId, &data); err != nil {
			return nil, err
		}
		var tradeState TradeState
		if err := json.Unmarshal([]byte(data), &tradeState); err != nil {
			return nil, err
		}
		tradeStates = append(tradeStates, tradeState)
	}
	return tradeStates, nil
}

func formatTimestamp(timestamp time.Time) string {
	return timestamp.UTC().Format("2006-01-02 15:04:05.999")
}

func formatJson(val interface{}) (string, error) {
	buf, err := json.Marshal(val)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
