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
	cronJob := time.NewTicker(time.Hour)

	run()

	for {
		select {
		case <-cronJob.C:
			run()
		}
	}
}

var existing = make(map[string]string)

// var existing string

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

	fmt.Printf("\"db started\": %v\n", "db started")

	err = db.Ping()
	if err != nil {
		fmt.Println("Error pinging the database")
		panic(err.Error())
	}

	check(db, "cosmos", configuration["cosmos"])
	check(db, "akash", configuration["akash"])
	check(db, "osmosis", configuration["osmosis"])
	check(db, "passage", configuration["passage"])
	check(db, "umee", configuration["umee"])
	check(db, "regen", configuration["regen"])
	check(db, "dydx", configuration["dydx"])
	check(db, "stargaze", configuration["stargaze"])
	check(db, "juno", configuration["juno"])
	check(db, "evmos", configuration["evmos"])
	check(db, "quasar", configuration["quasar"])
	check(db, "gravity", configuration["gravity"])
	check(db, "comdex", configuration["comdex"])
	check(db, "desmos", configuration["desmos"])
	check(db, "quicksilver", configuration["quicksilver"])
	check(db, "omniflix", configuration["omniflix"])
	check(db, "mars", configuration["mars"])
	check(db, "celestia", configuration["celestia"])
	check(db, "archway", configuration["archway"])
	check(db, "crescent", configuration["crescent"])

}

func check(db *sql.DB, entity string, config Configuration) {
	err := createTable(db, "validator."+entity)
	if err != nil {
		fmt.Println("Error creating table for entity:", err)
		return
	} else {
		// fmt.Println("RPC:", configuration["cosmos"].Rpc)
		// fmt.Println("Cosmos Address:", configuration["cosmos"].Cosmos_address)
		// Fetch token value from the endpoint
		// url := "https://api-cosmoshub-ia.cosmosia.notional.ventures/cosmos/staking/v1beta1/validators/cosmosvaloper1ddle9tczl87gsvmeva3c48nenyng4n56nghmjk"
		url := fmt.Sprintf("%s/cosmos/staking/v1beta1/validators/%s", config.Rpc, config.Cosmos_address)
		// fmt.Println("UUUUUUUUUUUUUUUUUUUUUUUUUUU", url)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println("Error fetching the URL:", err.Error())
			send("Unable to reach the endpoint for  " + entity)
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
		//a := fmt.Sprintf("The value of the validator token from the %s endpoint is %d", entity, v.Validators.Tokens)
		//fmt.Println(a)
		// fmt.Sprintf("The value of the validator token from the %s  endpoint is", entity)
		var temptoken string
		query := fmt.Sprintf("SELECT token FROM validator.%s ORDER BY id DESC LIMIT 1", entity)
		err = db.QueryRow(query).Scan(&temptoken)

		// err = db.QueryRow("SELECT token FROM validator.Cosmos ORDER BY id DESC LIMIT 1").Scan(&temptoken)
		if err != nil && err != sql.ErrNoRows {
			fmt.Println("Error getting data from the database", err.Error())
		}
		existing[entity] = temptoken
		if existing[entity] == "" {
			existing[entity] = "0"
		}
		existingBigInt := new(big.Int)
		existingBigInt, exit := existingBigInt.SetString(existing[entity], 10)
		if !exit {
			fmt.Println("Error converting data from database to *big.Int")
			return
		}
		if existingBigInt == nil || existingBigInt.Cmp(tokensBigInt) != 0 {
			query := fmt.Sprintf("INSERT INTO validator.%s (token) VALUES (?)", entity)
			_, err := db.Exec(query, v.Validators.Tokens)
			// _, err := db.Exec("INSERT INTO validator.Cosmos (token) VALUES (?)", v.Validators.Tokens)
			if err != nil {
				fmt.Println("Error inserting new value into the database:", err.Error())
				return
			}
			send("The voting power for " + entity + " has changed to " + v.Validators.Tokens)

		}

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

func createTable(db *sql.DB, entity string) error {
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INT AUTO_INCREMENT PRIMARY KEY, token VARCHAR(255))", entity)
	_, err := db.Exec(query)
	return err
}
