package main

import (
	"context"
	"log"
	"time"

	pb "appointment-service/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	log.Println("[Appointment Service] Menginisialisasi koneksi gRPC ke Pharmacy Service...")

	// Koneksi gRPC ke Pharmacy Service
	conn, err := grpc.Dial(
		"10.53.73.179:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Tidak dapat membentuk koneksi ke Pharmacy Service: %v", err)
	}
	defer conn.Close()

	client := pb.NewPharmacyServiceClient(conn)

	// Skenario pengujian:
	// 1. MED-AMX-500 dengan jumlah 10000 -> ResourceExhausted
	// 2. MED-TIDAK-ADA dengan jumlah 10 -> NotFound

	requestValid := &pb.CheckDrugRequest{
		DrugCode:       "MED-TIDAK-ADA",
		QuantityNeeded: 10,
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Second,
	)
	defer cancel()

	log.Printf(
		"[RPC Call] Memvalidasi obat: %s, jumlah: %d",
		requestValid.GetDrugCode(),
		requestValid.GetQuantityNeeded(),
	)

	resp, err := client.CheckDrugAvailability(ctx, requestValid)

	// Penanganan error gRPC
	if err != nil {
		st, ok := status.FromError(err)

		if !ok {
			log.Printf(
				"[ERROR] Terjadi kesalahan komunikasi: %v",
				err,
			)
			return
		}

		switch st.Code() {
		case codes.NotFound:
			log.Printf(
				"[PERINGATAN KLINIS] Kode obat tidak ditemukan: %s",
				st.Message(),
			)

		case codes.ResourceExhausted:
			log.Printf(
				"[PERINGATAN KLINIS] Permintaan melebihi kuota farmasi: %s",
				st.Message(),
			)

		default:
			log.Printf(
				"[ERROR] RPC Gagal: Code=%s, Message=%s",
				st.Code(),
				st.Message(),
			)
		}

		return
	}

	// Jika permintaan berhasil
	log.Println("================== HASIL RESPON gRPC ==================")
	log.Printf("Kode Obat      : %s", resp.GetDrugCode())
	log.Printf("Tersedia       : %t", resp.GetIsAvailable())
	log.Printf("Stok Aktual    : %d", resp.GetCurrentStock())
	log.Printf("Harga Satuan   : Rp %.2f", resp.GetUnitPrice())
	log.Printf("Catatan Sistem : %s", resp.GetMessage())
	log.Println("========================================================")
}