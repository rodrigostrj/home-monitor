#include <Arduino.h>
#include <DHT.h>
#include "secrets.h"

#define DHT_PIN  4
#define DHT_TYPE DHT22

DHT dht(DHT_PIN, DHT_TYPE);

void setup() {
    Serial.begin(115200);
    dht.begin();
    Serial.println("Home Monitor firmware starting...");
}

void loop() {
    float temperature = dht.readTemperature();
    float humidity    = dht.readHumidity();

    if (isnan(temperature) || isnan(humidity)) {
        Serial.println("Failed to read from DHT sensor");
    } else {
        Serial.printf("Temperature: %.1f °C  Humidity: %.1f %%\n",
                      temperature, humidity);
    }

    delay(30000); // 30-second interval
}
