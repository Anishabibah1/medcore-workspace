package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "pharmacy-service/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	conn, err := grpc.Dial(
		"localhost:50051",
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("Gagal koneksi ke Pharmacy Service: %v", err)
	}
	defer conn.Close()

	client := pb.NewPharmacyServiceClient(conn)

	// Timeout hanya 500 milidetik
	ctx, cancel := context.WithTimeout(
		context.Background(),
		500*time.Millisecond,
	)
	defer cancel()

	fmt.Println("==============================================")
	fmt.Println("TEST STEP 3.4 - RPC DEADLINE TIMEOUT")
	fmt.Println("==============================================")
	fmt.Println("Timeout Client : 500 ms")
	fmt.Println("Delay Server   : 2 detik")
	fmt.Println("Mengirim RPC CheckDrugAvailability...")
	fmt.Println()

	start := time.Now()

	response, err := client.CheckDrugAvailability(
		ctx,
		&pb.CheckDrugRequest{
			DrugCode:      "MED-AMX-500",
			QuantityNeeded: 10,
		},
	)

	elapsed := time.Since(start)

	if err != nil {
		code := status.Code(err)

		fmt.Println("==============================================")
		fmt.Println("HASIL RPC")
		fmt.Println("==============================================")
		fmt.Printf("Status Code : %s\n", code)
		fmt.Printf("Pesan       : %s\n", err)
		fmt.Printf("Waktu       : %v\n", elapsed)
		fmt.Println("==============================================")

		if code == codes.DeadlineExceeded {
			fmt.Println("SUCCESS: RPC mengalami DeadlineExceeded")
		}

		return
	}

	fmt.Println("RPC berhasil:")
	fmt.Println(response)
}