package main

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"sync"
	"tcp_server/teltonika_decoder"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/streadway/amqp"
)

type ConnectionInfo struct {
	Conn net.Conn
	IMEI int64
	// ProjectUUID string
}

type SensorData struct {
	IMEI int64
	// ProjectUUID string
	ParsedData string
	Timestamp  string
}

// Struct Json Teltonika
type InnerData struct {
	Avl_data_count int    `json:"Avl_data_count"`
	Avl_data_len   int    `json:"Avl_data_len"`
	CRC_16         int    `json:"CRC_16"`
	CODEC_ID       int    `json:"Codec_id"`
	Preamble       int    `json:"Preamble"`
	Response       string `json:"Response"`
	// Add other fields as needed
}

type Element struct {
	Element_id  int    `json:"Element_id"`
	Element_len int    `json:"Element_len"`
	Element_val string `json:"Element_val"`
}

type AVLData struct {
	Altitude      int32                    `json:"Altitude"`
	Angle         int32                    `json:"Angle"`
	Element_count int                      `json:"Element_count"`
	Event_id      int                      `json:"Event_id"`
	IO_elements   []map[string]interface{} `json:"IO_elements"`
	Latitude      int32                    `json:"Latitude"`
	Longitude     int32                    `json:"Longitude"`
	Priority      int                      `json:"Priority"`
	Satellite     int                      `json:"Satellite"`
	Speed         int                      `json:"Speed"`
	Timestamp     float64                  `json:"Timestamp"`
}

type Data struct {
	Avl_data []AVLData `json:"Avl_data"`
}

type Message struct {
	Data Data `json:"data"`
}

// func isIMEIValid(db *sql.DB, imei int64) (bool, string, error) {
// 	query := "SELECT project_uuid FROM devices WHERE imei = ?"
// 	var projectUUID string
// 	err := db.QueryRow(query, imei).Scan(&projectUUID)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return false, "", nil
// 		}
// 		return false, "", err
// 	}

// 	// IMEI found, return project_uuid
// 	return true, projectUUID, nil
// }

func cleanAndParseIMEI(imeiString string) (int64, error) {
	// Menghapus karakter escape Unicode ("\x00" dan "\x0f") dari string IMEI
	regex := regexp.MustCompile(`[^\x20-\x7E]+`)
	cleanedIMEI := regex.ReplaceAllString(imeiString, "")

	// Mengonversi string IMEI yang sudah dibersihkan menjadi nilai numerik
	numericIMEI, err := strconv.ParseInt(cleanedIMEI, 10, 64)
	if err != nil {
		return 0, err
	}

	return numericIMEI, nil
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		os.Exit(1)
	}
}

func sendToMQTT(jsonData SensorData, logFile *os.File) {
	fmt.Println(jsonData)
}

func declareQueue(ch *amqp.Channel, imei int64) (amqp.Queue, error) {
	queueName := strconv.FormatInt(imei, 10)

	// Declare a queue with the given name
	q, err := ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		// If the queue already exists, the error might be due to that, so ignore it
		if !isQueueExistError(err) {
			return q, err
		}
	}

	return q, nil
}

// isQueueExistError checks if the error indicates that the queue already exists.
func isQueueExistError(err error) bool {
	const queueExistErrorCode = 406
	amqpError, ok := err.(*amqp.Error)
	return ok && amqpError.Code == queueExistErrorCode
}

func createLogFile() (*os.File, error) {
	// Specify the folder name
	folderName := "logs"

	// Check if the folder already exists
	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		// Create the folder
		err := os.Mkdir(folderName, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	// Get the current date
	currentDate := time.Now().Format("2006-01-02")

	// Create the log file name with the specified format
	fileName := fmt.Sprintf("%s/log-%s.log", folderName, currentDate)

	// Open or create the log file
	logFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		return nil, err
	}

	return logFile, nil
}

func acceptConnections(listener net.Listener, incomingConn chan ConnectionInfo) {
	defer close(incomingConn)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			return
		}

		buffer := make([]byte, 2000)
		n, err := conn.Read(buffer)
		if err != nil {
			log.Println("Error reading from connection:", err)
			conn.Close()
			continue
		}

		data := buffer[:n]
		if len(data) == 17 {
			imeiString := string(data)
			numericIMEI, err := cleanAndParseIMEI(imeiString)
			if err != nil {
				fmt.Println("Error parsing IMEI to numeric value:", err)
				return
			}

			imei := numericIMEI
			// valid, projectUUID, err := isIMEIValid(db, imei)
			// if err != nil {
			// 	fmt.Println("Error checking IMEI validity:", err)
			// 	return
			// }
			// if !valid {
			// 	fmt.Println("Invalid IMEI")
			// 	return
			// }

			// fmt.Println("Imei Confirmed")
			if _, err := conn.Write([]byte{0x01}); err != nil {
				log.Println("Error writing to connection:", err)
				break
			}

			connectionInfo := ConnectionInfo{
				Conn: conn,
				IMEI: imei,
				// ProjectUUID: projectUUID,
			}

			incomingConn <- connectionInfo
		}
	}
}

func formattedTime(t time.Time) string {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		// Handle error
		return ""
	}
	return t.In(loc).Format("2006-01-02 15:04:05")
}

func processConnections(incomingConn chan ConnectionInfo, processedData chan SensorData, wg *sync.WaitGroup) {
	defer wg.Done()

	for conn := range incomingConn {
		buffer := make([]byte, 2000)
		n, err := conn.Conn.Read(buffer)
		if err != nil {
			log.Println("Error reading from connection:", err)
			conn.Conn.Close()
			continue
		}

		data := buffer[:n]
		argument := fmt.Sprintf("%x", data)
		bs, _ := hex.DecodeString(argument)
		parsedData, err := teltonika_decoder.Decode(&bs)
		avlDatacount := parsedData.Avl_data_count
		avlDatacounthex := fmt.Sprintf("%X", avlDatacount)
		//Send response using avl data count
		conn.Conn.Write([]byte(avlDatacounthex))

		jsonData, err := json.Marshal(parsedData)
		if err != nil {
			log.Println("Error encoding JSON data:", err)
			continue
		}

		sensorData := SensorData{
			IMEI: conn.IMEI,
			// ProjectUUID: conn.ProjectUUID,
			ParsedData: string(jsonData),
			Timestamp:  formattedTime(time.Now()),
		}

		processedData <- sensorData

		// Close the connection after processing data
		conn.Conn.Close()
	}

}

func respondToClients(processedData chan SensorData, logFile *os.File) {
	for data := range processedData {
		// fmt.Printf("IMEI: %d, ProjectUUID: %s\n", data.IMEI, data.ProjectUUID)
		sendToMQTT(data, logFile)
	}
}

func main() {
	config, err := LoadConfig()
	if err != nil {
		fmt.Println("Error setup config:", err)
		return
	}

	logFile, err := createLogFile()
	failOnError(err, "Failed create a log file")
	defer logFile.Close()

	listener, err := net.Listen("tcp", config.TCP.URL)
	if err != nil {
		fmt.Println("Error starting the server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server listening port on : " + config.TCP.URL)

	var wg sync.WaitGroup

	incomingConn := make(chan ConnectionInfo, 1000)
	processedData := make(chan SensorData, 1000)

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config.DB.Username, config.DB.Password, config.DB.Host, config.DB.Port, config.DB.Name))
	failOnError(err, "Failed to connect to the database")
	defer db.Close()

	// Menerima koneksi asinkron
	go acceptConnections(listener, incomingConn)

	// Multiple Device on One TCP (Make 5 Channel)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go processConnections(incomingConn, processedData, &wg)
	}

	// Tahap 3: Merespons ke klien secara asinkron
	go respondToClients(processedData, logFile)

	wg.Wait()
}
