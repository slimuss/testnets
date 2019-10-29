package src

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"text/tabwriter"
	"uptime/db"

	"gopkg.in/mgo.v2/bson"
)

type handler struct {
	db db.DB
}

func New(db db.DB) handler {
	return handler{db}
}

func (h handler) CalculateUptime(startBlock int, endBlock int) {
	var validatorsList []ValidatorInfo //Intializing validators uptime

	fmt.Println("Fetching blocks from:", startBlock, ", to:", endBlock)

	//Read all blocks
	blocks, _ := h.db.FetchBlocks(startBlock, endBlock)
	numBlocks := len(blocks)

	fmt.Println("Fetched ", numBlocks, " blocks. Calculating uptime ...")

	for currentHeight := 0; currentHeight < numBlocks; currentHeight++ {
		for _, valAddr := range blocks[currentHeight].Validators {

			//Get validator address from existed validator uptime count
			index := GetValidatorIndex(valAddr, validatorsList)

			if index > 0 {
				// If validator is present in the list already (i.e., joined the network in previous block heights)
				// Update uptime details
				validatorsList[index].Info.UptimeCount++
			} else {
				// If the validator is not present in the list i.e., newly joined in the current block
				// Fetch Validator information and Push to validators list
				// Initialize the validator uptime info with default info (i.e., 1)

				query := bson.M{
					"address": valAddr,
				}

				//Get validator by using validator address
				validator, _ := h.db.GetValidator(query)

				valAddressInfo := ValidatorInfo{
					ValAddress: valAddr,
					Info: Info{
						UptimeCount:  1,
						Moniker:      validator.Description.Moniker,
						OperatorAddr: validator.OperatorAddress,
						StartBlock:   int64(currentHeight),
					},
				}

				//Inserting new validator into uptime count
				validatorsList = append(validatorsList, valAddressInfo)
			}
		}
	}

	//Printing Uptime results in tabular view
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 0, ' ', tabwriter.Debug)
	fmt.Fprintln(w, " Address\t Moniker\t Uptime Count")

	for _, data := range validatorsList {
		fmt.Fprintln(w, " "+data.ValAddress+"\t "+data.Info.Moniker+"\t  "+strconv.Itoa(int(data.Info.UptimeCount)))
	}

	w.Flush()

	//Exporing into csv file
	ExportIntoCsv(validatorsList)
}

// GetValidatorIndex
// returns the index of the validator from the list
func GetValidatorIndex(validatorAddr string, validatorsList []ValidatorInfo) int {
	var pos int

	for index, addr := range validatorsList {
		if addr.ValAddress == validatorAddr {
			pos = index
		}
	}

	return pos
}

func ExportIntoCsv(data []ValidatorInfo) {
	Header := []string{
		"Address", "Moniker", "Uptime Count",
	}

	file, err := os.Create("result.csv")

	if err != nil {
		log.Fatal("Cannot write to file", err)
	}

	defer file.Close() //Close file

	writer := csv.NewWriter(file)

	defer writer.Flush()

	//Write header titles
	_ = writer.Write(Header)

	for _, record := range data {
		uptimeCount := strconv.Itoa(int(record.Info.UptimeCount))
		addrObj := []string{record.ValAddress, record.Info.Moniker, uptimeCount}
		err := writer.Write(addrObj)

		if err != nil {
			log.Fatal("Cannot write to file", err)
		}
	}
}
