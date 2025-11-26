package main

import (
	"github.com/rs/zerolog/log"
)

func checkToolConnectivity(cfg *Config) {
	if cfg.Kafka.Enabled {
		if err := checkKafkaConnection(cfg.Kafka); err != nil {
			log.Error().
				Err(err).
				Strs("brokers", cfg.Kafka.Brokers).
				Msg("Failed to connect to Kafka")
		} else {
			log.Info().
				Strs("brokers", cfg.Kafka.Brokers).
				Msg("Successfully connected to Kafka")
		}
	}

	if cfg.MySQL.Enabled {
		if err := checkMySQLConnection(cfg.MySQL); err != nil {
			log.Error().
				Err(err).
				Str("host", cfg.MySQL.Host).
				Str("port", cfg.MySQL.Port).
				Str("database", cfg.MySQL.Database).
				Msg("Failed to connect to MySQL")
		} else {
			log.Info().
				Str("host", cfg.MySQL.Host).
				Str("port", cfg.MySQL.Port).
				Str("database", cfg.MySQL.Database).
				Msg("Successfully connected to MySQL")
		}
	}

	if cfg.ClickHouse.Enabled {
		if err := checkClickHouseConnection(cfg.ClickHouse); err != nil {
			log.Error().
				Err(err).
				Str("host", cfg.ClickHouse.Host).
				Str("port", cfg.ClickHouse.Port).
				Str("database", cfg.ClickHouse.Database).
				Msg("Failed to connect to ClickHouse")
		} else {
			log.Info().
				Str("host", cfg.ClickHouse.Host).
				Str("port", cfg.ClickHouse.Port).
				Str("database", cfg.ClickHouse.Database).
				Msg("Successfully connected to ClickHouse")
		}
	}
}
