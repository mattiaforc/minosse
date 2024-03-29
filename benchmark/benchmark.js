import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
    vus: 100,
    duration: '60s',
};

export default function () {
    let res = http.get('http://localhost:8080/index.html')
    check(res, {"Response code 200": (res) => res.status === 200})
}
