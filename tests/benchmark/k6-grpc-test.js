import grpc from 'k6/net/grpc';
import { check } from 'k6';

const client = new grpc.Client();

client.load(['../../proto'], 'pharmacy.proto');

export const options = {
  stages: [
    { duration: '10s', target: 50 },
    { duration: '20s', target: 200 },
    { duration: '10s', target: 0 },
  ],
};

export default function () {
  client.connect('10.53.73.179:50051', {
    plaintext: true,
  });

  const response = client.invoke(
    'medcore.pharmacy.v1.PharmacyService/CheckDrugAvailability',
    {
      drug_code: 'MED-AMX-500',
      quantity_needed: 10,
    }
  );

  check(response, {
    'gRPC status is OK': (r) =>
      r && r.status === grpc.StatusOK,

    'drug is available': (r) =>
      r &&
      r.status === grpc.StatusOK &&
      r.message.isAvailable === true,
  });

  client.close();
}