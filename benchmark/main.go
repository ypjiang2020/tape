package main

import (
	"fmt"
	"log"

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
			log.Printf(viper.GetString("config"))
			log.Println(viper.GetString("channel"))
			log.Println(viper.Get("endorsers"))
			log.Println(viper.Get("committer"))
		},
	}
	initCmd = &cobra.Command{
		Use:   "init",
		Short: "initialize blockchain state",
		Run: func(cmd *cobra.Command, args []string) {
			client.RunInitCmd(config)
		},
	}
	transactionCmd = &cobra.Command{
		Use:   "txn",
		Short: "send transactions",
		Run: func(cmd *cobra.Command, args []string) {
			client.RunTxnCmd(config)

		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringP("config", "c", "config.yaml", "config file")
	rootCmd.PersistentFlags().IntP("interval", "i", 10, "benchmark time in seconds, default 10s")
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("interval", rootCmd.PersistentFlags().Lookup("interval"))
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
