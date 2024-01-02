package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
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
	Status string `json:"status"`
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

		url := fmt.Sprintf("%s/cosmos/staking/v1beta1/validators/%s", config.Rpc, config.Cosmos_address)
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
		f_number := v.Validators.Tokens
		f_number = strings.TrimSpace(f_number)
		float, success := new(big.Float).SetString(f_number)
		if !success {
			fmt.Println("Error converting string to big.Float")
			return
		}
		divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil))
		result := new(big.Float).Quo(float, divisor)

		var temptoken string
		var existingStatus string
		s := v.Validators.Status
		if s == "BOND_STATUS_UNBONDED" {
			send(entity + " validator is in state : BOND_STATUS_UNBONDED")
		}
		query := fmt.Sprintf("SELECT token,status FROM validator.%s ORDER BY id DESC LIMIT 1", entity)
		err = db.QueryRow(query).Scan(&temptoken, &existingStatus)
		if err != nil {
			if err == sql.ErrNoRows {
				insertQuery := fmt.Sprintf("INSERT INTO validator.%s (token,status) VALUES (?,?)", entity)
				_, err := db.Exec(insertQuery, result.Text('f', 6), s)

				if err != nil {
					fmt.Println("Error inserting new value into the database:", err.Error())
					return
				}

				return
			} else {
				fmt.Println("Error:", err.Error())
				return
			}
		}

		existing[entity] = temptoken
		existingBigFloat := new(big.Float)
		existingBigFloat, exit := existingBigFloat.SetString(existing[entity])

		if !exit {
			fmt.Println("Error converting data from database to *big.Float")
			return
		}

		if existingBigFloat.Cmp(result) != 0 || existingStatus != s {
			if existingBigFloat.Cmp(result) < 0 {
				send(entity + " voting power has increased from " + existingBigFloat.String() + " to " + result.String())
			}
			if existingBigFloat.Cmp(result) > 0 {
				send(entity + " voting power has decreased from " + existingBigFloat.String() + " to " + result.String())
			}
			updateQuery := fmt.Sprintf("UPDATE validator.%s SET token = ?,status = ? ORDER BY id DESC LIMIT 1", entity)
			_, err := db.Exec(updateQuery, result.Text('f', 6), s)
			if err != nil {
				fmt.Println("Error updating database:", err)
				return
			}

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
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INT AUTO_INCREMENT PRIMARY KEY, token VARCHAR(255),status VARCHAR(30))", entity)
	_, err := db.Exec(query)
	return err
}
