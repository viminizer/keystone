// Clients that use keystone like an app uses a cache. Each request reads a
// key. On a miss, the app loads the value from its database and saves it
// with a time limit. Sometimes the app changes the data and deletes the key.
//
// A few keys are very popular and most are rarely used, like real users or
// products. The load goes up and down every 10 minutes, like a busy and a
// quiet time of day. It runs for a week, or until it is stopped.
//
//   k6 run observability/load.js

import redis from "k6/experimental/redis";

const client = new redis.Client("redis://localhost:6380");

const keyCount = 100000;
const cycleMinutes = 10;
const quietRate = 500;
const busyRate = 3000;

function makeStages() {
  const stages = [];
  const minutesInAWeek = 7 * 24 * 60;
  for (let minute = 0; minute < minutesInAWeek; minute++) {
    const angle = (2 * Math.PI * minute) / cycleMinutes;
    const busy = (1 - Math.cos(angle)) / 2;
    stages.push({ duration: "1m", target: Math.round(quietRate + busy * (busyRate - quietRate)) });
  }
  return stages;
}

export const options = {
  scenarios: {
    cache: {
      executor: "ramping-arrival-rate",
      startRate: quietRate,
      timeUnit: "1s",
      preAllocatedVUs: 50,
      maxVUs: 500,
      stages: makeStages(),
    },
  },
};

// Taking a random number to a power makes small numbers much more likely,
// so the first keys get most of the requests.
function popularKey() {
  return "user:" + Math.floor(keyCount * Math.pow(Math.random(), 4));
}

function randomValue() {
  const length = 50 + Math.floor(Math.random() * 950);
  return "x".repeat(length);
}

function randomSeconds() {
  return 10 + Math.floor(Math.random() * 290);
}

export default async function () {
  const key = popularKey();

  if (Math.random() < 0.03) {
    await client.del(key);
    return;
  }

  try {
    await client.get(key);
  } catch (error) {
    // A miss: load the value from the "database" and cache it.
    await client.set(key, randomValue(), randomSeconds());
  }
}
