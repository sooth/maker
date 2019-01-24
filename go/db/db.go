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

package db

import (
	"fmt"
	"database/sql"
	"gitlab.com/crankykernel/maker/go/types"
	"time"
	"encoding/json"
	"gitlab.com/crankykernel/maker/go/log"
	"strings"
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

	if version < 3 {
		rows, err := tx.Query(`select id, data from binance_trade`)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to load trades: %v", err)
		}

		count := 0
		for rows.Next() {
			var localId string
			var data string
			if err := rows.Scan(&localId, &data); err != nil {
				log.WithError(err).Error("Failed to scan row.")
				continue
			}

			var tradeState0 types.TradeStateV0
			if err := json.Unmarshal([]byte(data), &tradeState0); err != nil {
				log.WithError(err).Error("Failed to unmarshal v0 trade state.")
				continue
			}

			if tradeState0.Version > 0 {
				continue
			}

			tradeState := types.TradeStateV0ToTradeStateV1(tradeState0)
			TxDbUpdateTradeState(tx, &tradeState)
			count += 1
		}
		log.Printf("Migrated %d trades from v0 to v1.", count)
		if err := incrementVersion(tx, 3); err != nil {
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

func DbSaveBinanceRawExecutionReport(timestamp time.Time, event []byte) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`insert into binance_raw_execution_report (timestamp, report) values (?, ?)`,
		formatTimestamp(timestamp), event)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func DbSaveTrade(trade *types.Trade) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	data, err := formatJson(trade.State)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`insert into binance_trade (id, data) values (?, ?)`,
		trade.State.TradeID, data)
	tx.Commit()
	return err
}

func TxDbUpdateTradeState(tx *sql.Tx, trade *types.TradeState) error {
	data, err := formatJson(trade)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`update binance_trade set data = ? where id = ?`,
		data, trade.TradeID)
	return err
}

func DbUpdateTrade(trade *types.Trade) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	err = TxDbUpdateTradeState(tx, &trade.State)
	if err != nil {
		log.WithError(err).Error("Failed to update trade to DB.")
		tx.Rollback()
	} else {
		tx.Commit()
	}
	return err
}

func DbArchiveTrade(trade *types.Trade) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`update binance_trade set archived = 1 where id = ?`,
		trade.State.TradeID)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return err
}

func DbRestoreTradeState() ([]types.TradeState, error) {
	rows, err := db.Query(`select id, data from binance_trade where archived = 0`)
	if err != nil {
		return nil, err
	}

	tradeStates := []types.TradeState{}

	for rows.Next() {
		var localId string
		var data string
		if err := rows.Scan(&localId, &data); err != nil {
			return nil, err
		}
		var tradeState types.TradeState
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

func DbGetTradeByID(tradeId string) (*types.TradeState, error) {
	row := db.QueryRow(
		`select data from binance_trade where id = ?`,
		tradeId)
	var data string
	err := row.Scan(&data)
	if err != nil {
		return nil, err
	}
	var tradeState types.TradeState
	err = json.Unmarshal([]byte(data), &tradeState)
	if err != nil {
		return nil, err
	}
	return &tradeState, nil
}

type TradeQueryOptions struct {
	IsClosed bool
}

func DbQueryTrades(options TradeQueryOptions) ([]types.TradeState, error) {

	where := []string{}

	if options.IsClosed {
		where = append(where, fmt.Sprintf("json_extract(binance_trade.data, '$.CloseTime') != ''"))
	}

	sql := "select id, data from binance_trade"
	if len(where) > 0 {
		sql = fmt.Sprintf("%s WHERE %s", sql, strings.Join(where, "AND "))
	}

	rows, err := db.Query(sql)
	if err != nil {
		return nil, err
	}

	tradeStates := []types.TradeState{}

	for rows.Next() {
		var localId string
		var data string
		if err := rows.Scan(&localId, &data); err != nil {
			return nil, err
		}
		var tradeState types.TradeState
		if err := json.Unmarshal([]byte(data), &tradeState); err != nil {
			return nil, err
		}
		tradeStates = append(tradeStates, tradeState)
	}
	return tradeStates, nil
}
