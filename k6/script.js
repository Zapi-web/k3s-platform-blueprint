import { check, sleep } from 'k6';
import http from 'k6/http';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';

export const options = {
  stages: [
    { duration: '30s', target: 100 },
    { duration: '1m',  target: 400 },
    { duration: '1m',  target: 800 },
    { duration: '2m',  target: 1000 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
    'http_req_duration{name:PostCreateLink}': ['p(95)<400'],
    'http_req_duration{name:GetRedirectLink}': ['p(95)<100'],
  },
};

export default function() {
    const payload = JSON.stringify({
        url: "https://google.com",
        user_id: 1
    });

    const params = {
        headers: { 'Content-Type': 'application/json' },
        tags: { name: 'PostCreateLink' }
    };

    const postRes = http.post(`${BASE_URL}/api/v1/`, payload, params);

    const isPostOk = check(postRes, {
        'POST link created (200)': (r) => r.status === 200
    });

    if (isPostOk && postRes.body) {
        let shortURL;
        try {
            shortURL = postRes.json('short_url');
        } catch (_) {}

        if (shortURL) {
            const getParams = {
                redirects: 0,
                tags: { name: 'GetRedirectLink' }
            };

            const getRes = http.get(http.url`${BASE_URL}/${shortURL}`, getParams);

            check(getRes, {
                'GET short link received (302)': (r) => r.status === 302,
                'Redirect location is Google': (r) =>
                    r.headers['Location'] === 'https://google.com' ||
                    r.headers['location'] === 'https://google.com',
            });
        }
    }

    sleep(0.05);
}
