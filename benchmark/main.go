package main

import (
	"log"
	"fmt"
	"github.com/Yunpeng-J/tape/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	config  client.Config
	rootCmd = &cobra.Command{
		Use:   "benchmark",
		Short: "A close loop client",
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("TODO")
		},
	}
	initCmd = &cobra.Command{
		Use:   "init",
		Short: "initialize blockchain state",
		Run: func(cmd *cobra.Command, args []string) {
			viper.SetDefault("transactionType", "init")
			client.RunInitCmd(config)
		},
	}
	transactionCmd = &cobra.Command{
		Use:   "txn",
		Short: "send transactions",
		Run: func(cmd *cobra.Command, args []string) {
			viper.SetDefault("transactionType", "txn")
			client.RunTxnCmd(config)
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringP("config", "", "config.yaml", "config file")
	rootCmd.PersistentFlags().StringP("workload", "", "smallbank", "the type of workload")
	rootCmd.PersistentFlags().IntP("interval", "", 10, "benchmark time in seconds, default 10s")
	rootCmd.PersistentFlags().IntP("clientsPerEndorser", "", 1, "the number of clients for one endorser")
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("interval", rootCmd.PersistentFlags().Lookup("interval"))
	viper.BindPFlag("clientsPerEndorser", rootCmd.PersistentFlags().Lookup("clientsPerEndorser"))
	viper.BindPFlag("workload", rootCmd.PersistentFlags().Lookup("workload"))
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(transactionCmd)

}

func initConfig() {
	conf := viper.GetString("config")
	viper.SetConfigFile(conf)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("loading config file %s failed", conf))
	}
	var err error
	config, err = client.LoadConfig(conf)
	if err != nil {
		panic(fmt.Sprintf("loading config file %s failed: %v", conf, err))
	}
}

func main() {
	rootCmd.Execute()
}
