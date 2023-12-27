package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/spf13/viper"

	_ "github.com/go-sql-driver/mysql"
)

type Configuration struct {
	Rpc            string `yaml:"rpc"`
	Chain_Id       string `yaml:"chain_id"`
	Cosmos_address string `yaml:"cosmos_address"`
}

type Validator struct {
	Tokens string `json:"tokens"`
}
type VResponse struct {
	Validators Validator `json:"validator"`
}

func main() {
	cronJob := time.NewTicker(time.Minute)

	run()

	for {
		select {
		case <-cronJob.C:
			run()
		}
	}
}

var existing string

func run() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetConfigType("yml")
	var configuration map[string]Configuration
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}
	err := viper.Unmarshal(&configuration)
	if err != nil {
		fmt.Printf("Unable to decode into struct, %v", err)
	}
	// ////////////////////////////////////////////////////////////////////////////////////////////

	db, err := sql.Open("mysql", "root:vitwit@tcp(localhost:3306)/validator")
	if err != nil {
		fmt.Println("ERR")
		panic(err.Error())
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Println("Error pinging the database")
		panic(err.Error())
	}
	// fmt.Println("RPC:", configuration["cosmos"].Rpc)
	// fmt.Println("Cosmos Address:", configuration["cosmos"].Cosmos_address)
	// Fetch token value from the endpoint
	// url := "https://api-cosmoshub-ia.cosmosia.notional.ventures/cosmos/staking/v1beta1/validators/cosmosvaloper1ddle9tczl87gsvmeva3c48nenyng4n56nghmjk"
	url := fmt.Sprintf("%s/cosmos/staking/v1beta1/validators/%s", configuration["cosmos"].Rpc, configuration["cosmos"].Cosmos_address)
	// fmt.Println("UUUUUUUUUUUUUUUUUUUUUUUUUUU", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching the URL:", err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading the unbounded body", err.Error())
		return
	}

	var v VResponse
	err = json.Unmarshal(body, &v)
	if err != nil {
		fmt.Println("Error while unmarshalling the data ", err.Error())
		return
	}
	tokensBigInt := new(big.Int)
	tokensBigInt, success := tokensBigInt.SetString(v.Validators.Tokens, 10)
	if !success {
		fmt.Println("Error converting string to *big.Int")
		return
	}
	fmt.Println("The value of the validator token from the endpoint is", v.Validators.Tokens)
	err = db.QueryRow("SELECT token FROM validator.Cosmos ORDER BY id DESC LIMIT 1").Scan(&existing)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error getting data from the database", err.Error())
	}
	if existing == "" {
		existing = "0"
	}
	existingBigInt := new(big.Int)
	existingBigInt, exit := existingBigInt.SetString(existing, 10)
	if !exit {
		fmt.Println("Error converting data from database to *big.Int")
		return
	}
	if existingBigInt == nil || existingBigInt.Cmp(tokensBigInt) != 0 {
		_, err := db.Exec("INSERT INTO validator.Cosmos (token) VALUES (?)", v.Validators.Tokens)
		if err != nil {
			fmt.Println("Error inserting new value into the database:", err.Error())
			return
		}
		send("The voting power has changed to " + v.Validators.Tokens)

	}

}
func send(message string) {
	botToken := "6780687251:AAFoZtSIjXgcmn3tXd7HRbW86sn0rgpLmTk"
	chatID := int64(-1002016620029)
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		fmt.Println(err.Error())
	}
	msg := tgbotapi.NewMessage(chatID, message)
	_, err = bot.Send(msg)
	if err != nil {
		fmt.Println(err.Error())
	}

}
