import http from 'k6/http';
import { check } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 50 },
    { duration: '20s', target: 200 },
    { duration: '10s', target: 0 },
  ],
};

export default function () {
  const url =
    'http://10.53.73.179:8081/api/v1/drugs/check?drug_code=MED-AMX-500';

  const res = http.get(url);

  check(res, {
    'status is 200': (r) => r.status === 200,
    'drug_code is correct': (r) => {
      try {
        return JSON.parse(r.body).drug_code === 'MED-AMX-500';
      } catch (e) {
        return false;
      }
    },
  });
}