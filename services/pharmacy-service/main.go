package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"

	pb "pharmacy-service/pb"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type pharmacyServer struct {
	pb.UnimplementedPharmacyServiceServer
	db *mongo.Database
}

// =========================================================
// gRPC: CheckDrugAvailability
// =========================================================
func (s *pharmacyServer) CheckDrugAvailability(
	ctx context.Context,
	req *pb.CheckDrugRequest,
) (*pb.CheckDrugResponse, error) {

	// ResourceExhausted jika permintaan > 5000
	if req.QuantityNeeded > 5000 {
		return nil, status.Errorf(
			codes.ResourceExhausted,
			"Permintaan melebihi batas kuota farmasi",
		)
	}

	var drug struct {
		DrugCode   string  `bson:"drug_code"`
		Name       string  `bson:"name"`
		UnitPrice  float64 `bson:"unit_price"`
		TotalStock int32   `bson:"total_stock"`
	}

	err := s.db.Collection("drugs").FindOne(
		ctx,
		bson.M{"drug_code": req.DrugCode},
	).Decode(&drug)

	// Obat tidak ditemukan
	if err == mongo.ErrNoDocuments {
		return nil, status.Errorf(
			codes.NotFound,
			"Obat dengan kode %s tidak ditemukan",
			req.DrugCode,
		)
	}

	// Error database lainnya
	if err != nil {
		log.Printf("[ERROR] Gagal mengambil data obat: %v", err)

		return nil, status.Errorf(
			codes.Internal,
			"Gagal mengambil data obat",
		)
	}

	isAvailable := drug.TotalStock >= req.QuantityNeeded

	message := "Stok obat tidak mencukupi"

	if isAvailable {
		message = "Stok obat mencukupi"
	}

	return &pb.CheckDrugResponse{
		DrugCode:     drug.DrugCode,
		IsAvailable:  isAvailable,
		CurrentStock: drug.TotalStock,
		UnitPrice:    drug.UnitPrice,
		Message:      message,
	}, nil
}

// =========================================================
// gRPC: ReservePrescriptionStock
// =========================================================
func (s *pharmacyServer) ReservePrescriptionStock(
	ctx context.Context,
	req *pb.ReserveStockRequest,
) (*pb.ReserveStockResponse, error) {

	return nil, status.Error(
		codes.Unimplemented,
		"ReservePrescriptionStock belum diimplementasikan",
	)
}

// =========================================================
// REST API: GET /api/v1/drugs/check
// =========================================================
func startRESTServer(mongoClient *mongo.Client) {

	http.HandleFunc("/api/v1/drugs/check", func(w http.ResponseWriter, r *http.Request) {

		// Hanya menerima GET
		if r.Method != http.MethodGet {
			http.Error(
				w,
				"Method not allowed",
				http.StatusMethodNotAllowed,
			)
			return
		}

		// Ambil drug_code dari query parameter
		drugCode := r.URL.Query().Get("drug_code")

		// Jika drug_code kosong
		if drugCode == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			json.NewEncoder(w).Encode(map[string]string{
				"error": "drug_code wajib diisi",
			})

			return
		}

		// Akses collection drugs
		coll := mongoClient.
			Database("medcore_pharmacy_db").
			Collection("drugs")

		// Struktur data obat
		var drug struct {
			DrugCode   string  `bson:"drug_code"`
			Name       string  `bson:"name"`
			UnitPrice  float64 `bson:"unit_price"`
			TotalStock int32   `bson:"total_stock"`
		}

		// Query MongoDB
		err := coll.FindOne(
			r.Context(),
			bson.M{"drug_code": drugCode},
		).Decode(&drug)

		// Jika obat tidak ditemukan
		if err != nil {

			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			w.WriteHeader(http.StatusNotFound)

			json.NewEncoder(w).Encode(
				map[string]string{
					"error": "Obat tidak ditemukan",
				},
			)

			return
		}

		// Response JSON
		w.Header().Set(
			"Content-Type",
			"application/json",
		)

		json.NewEncoder(w).Encode(
			map[string]interface{}{
				"drug_code":     drug.DrugCode,
				"is_available":  drug.TotalStock >= 10,
				"current_stock": drug.TotalStock,
				"unit_price":    drug.UnitPrice,
				"message":       "Stok obat mencukupi",
			},
		)
	})

	log.Println("MedCore Pharmacy REST Baseline aktif pada port :8081")

	// Jalankan REST Server
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Printf(
			"[ERROR] REST server berhenti: %v",
			err,
		)
	}
}

// =========================================================
// MAIN
// =========================================================
func main() {

	// =========================================================
	// 1. Koneksi MongoDB
	// =========================================================

	mongoURI := "mongodb://adm_pharmacy_svc:SecuredPassPharm2026!@localhost:27017/medcore_pharmacy_db?authSource=admin"

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	client, err := mongo.Connect(
		ctx,
		options.Client().ApplyURI(mongoURI),
	)

	if err != nil {
		log.Fatalf(
			"[ERROR] Gagal membuat koneksi MongoDB: %v",
			err,
		)
	}

	// Ping MongoDB
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf(
			"[ERROR] MongoDB tidak dapat diakses: %v",
			err,
		)
	}

	log.Println("[MongoDB] Koneksi berhasil")

	db := client.Database("medcore_pharmacy_db")

	// =========================================================
	// 2. Jalankan REST Server
	// =========================================================

	go startRESTServer(client)

	// =========================================================
	// 3. Jalankan gRPC Server
	// =========================================================

	lis, err := net.Listen("tcp", ":50051")

	if err != nil {
		log.Fatalf(
			"[ERROR] Gagal membuka port 50051: %v",
			err,
		)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterPharmacyServiceServer(
		grpcServer,
		&pharmacyServer{
			db: db,
		},
	)

	log.Println("==================================================")
	log.Println("MedCore Pharmacy gRPC Service aktif pada port :50051")
	log.Println("MedCore Pharmacy REST Baseline aktif pada port :8081")
	log.Println("==================================================")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf(
			"[ERROR] gRPC server berhenti: %v",
			err,
		)
	}
}