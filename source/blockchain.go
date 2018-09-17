package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

)

type Block struct {
	Index     int
	Timestamp string
	PatientInfo string
	ProblemList string
	ConsultationReports string
	TestResults string
	Hash      string
	PreviousHash  string
}

var Blockchain []Block

func calculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.PatientInfo) + string(block.ProblemList) + string(block.ConsultationReports) + string(block.TestResults) +block.PreviousHash
	hash := sha256.New()
	hash.Write([]byte(record))
	hashed := hash.Sum(nil)
	return hex.EncodeToString(hashed)
}

func generateBlock(oldBlock Block, PatientInfo string, ProblemList string, ConsultationReports string, TestResults string) (Block, error) {
	var newBlock Block
	current_time := time.Now()
	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = current_time.String()
	newBlock.PatientInfo = PatientInfo
	newBlock.ProblemList = ProblemList
	newBlock.ConsultationReports = ConsultationReports
	newBlock.TestResults = TestResults
	newBlock.PreviousHash = oldBlock.Hash
	newBlock.Hash = calculateHash(newBlock)
	return newBlock, nil
}

func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index + 1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PreviousHash {
		return false
	}

	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")
	return muxRouter
}

func run() error {
	mux := makeMuxRouter()
	httpAddr := os.Getenv("ADDR")
	log.Println("Listening on ", os.Getenv("ADDR"))
	s := &http.Server{
		Addr:           ":" + httpAddr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	error := s.ListenAndServe()
	if error != nil {
		return error
	}

	return nil
}

func respondWithFullBlockchain(write http.ResponseWriter) {
	bytes, error := json.MarshalIndent(Blockchain, "", "  ")   //JSON format can be wiewed by localhost:8080 (.env)
	if error != nil {
		http.Error(write, error.Error(), http.StatusInternalServerError)
		return
	}

	io.WriteString(write, "<!DOCTYPE HTML><html><head><title>My blockchain</title></head><body><pre>")
	io.WriteString(write, string(bytes))
	io.WriteString(write, "</pre>")
	io.WriteString(write, `<form action="" method="post">
	                           PatientInfo:<br><input type="text" name="PatientInfo"><br>
	                           ProblemList:<br><input type="text" name="ProblemList"><br>
	                           ConsultationReports:<br><input type="text" name="ConsultationReports"><br>
	                           TestResults:<br><input type="text" name="TestResults"><br>

	                           <input type="submit" value="Submit Data" />
	                        </form>`)
	io.WriteString(write,"</body></html>")
}

func handleGetBlockchain(write http.ResponseWriter, request *http.Request) {
	respondWithFullBlockchain(write)
}

type Message struct {
	PatientInfo string
	ProblemList string
	ConsultationReports string
	TestResults string
}

func respondWithJSON(write http.ResponseWriter, request *http.Request, code int, payload interface{}) {
	response, error := json.MarshalIndent(payload, "", "  ")
	if error != nil {
		write.WriteHeader(http.StatusInternalServerError)
		write.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}
	write.WriteHeader(code)
	write.Write(response)
}

func StringToMessage(post_request string) Message {
   var message Message
   var substrings = strings.Split(post_request, "&")
   for _, v := range substrings {
      var field_pair = strings.Split(v, "=")
      switch field_pair[0] {
         case "PatientInfo": message.PatientInfo = field_pair[1]
         case "ProblemList": message.ProblemList =  field_pair[1]
         case "ConsultationReports": message.ConsultationReports = field_pair[1]
         case "TestResults": message.TestResults = field_pair[1]
      }
   }

   return message
}

func handleWriteBlock(write http.ResponseWriter, request *http.Request) {
	body, error := ioutil.ReadAll(request.Body)
	var bodyAsString = string(body[:])
	var message Message = StringToMessage(bodyAsString)
	newBlock, error := generateBlock(Blockchain[len(Blockchain)-1], message.PatientInfo, message.ProblemList, message.ConsultationReports, message.TestResults)
	if error != nil {
		respondWithJSON(write, request, http.StatusInternalServerError, message)
		return
	}
	if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
		Blockchain = append(Blockchain, newBlock)
	}

	respondWithFullBlockchain(write)
}

func main() {
	error := godotenv.Load()
	if error != nil {
		log.Fatal(error)
	}

	current_time := time.Now()
	initialBlock := Block{0, current_time.String(),"", "", "", "", "", ""}
	spew.Dump(initialBlock)
	Blockchain = append(Blockchain, initialBlock)
	log.Fatal(run())

}
