use btleplug::api::PeripheralProperties;
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};
use uuid::Uuid;

const SERVICE_UUID_STR: &str = "0000fd3d-0000-1000-8000-00805f9b34fb";

pub trait SwitchBotDeviceScanner<T>
where
    T: Serialize,
{
    fn scan(properties: &PeripheralProperties) -> Result<Option<T>, Box<dyn std::error::Error>>;
}

#[derive(Debug, Serialize, Deserialize)]
pub struct MeterProCo2Data {
    pub device_type: String,  //	the type of the device
    pub device_mac: String,   //	the MAC address of the device
    pub temperature: f32,     //	the current temperature reading
    pub scale: String,        //	the current temperature unit being used
    pub humidity: u8,         //	the current humidity reading in percentage
    pub co2: u16,             //	CO2 ppm value, 0-9999
    pub battery: u8,          //	the current battery level, 0-100
    pub time_of_sample: u128, //	the time stamp when the event is sent
}

pub struct MeterProCo2Scanner {}

impl SwitchBotDeviceScanner<MeterProCo2Data> for MeterProCo2Scanner {
    fn scan(
        properties: &PeripheralProperties,
    ) -> Result<Option<MeterProCo2Data>, Box<dyn std::error::Error>> {
        let target_service_uuid = Uuid::parse_str(SERVICE_UUID_STR)?;
        if let Some(service_data) = properties.service_data.get(&target_service_uuid) {
            let battery = service_data[2] & 0b01111111;

            if let Some(manufacturer_data) = properties.manufacturer_data.values().next() {
                if manufacturer_data.len() != 16 {
                    return Ok(None);
                }

                let s_payload = &manufacturer_data[6..];
                let co2 = (s_payload[7] as u16) * 256 + (s_payload[8] as u16);
                let humidity = s_payload[4] & 0b01111111;
                let temp_int = s_payload[2] & 0b00001111;
                let temp_dec = s_payload[3] & 0b01111111;
                let temperature = (temp_int as f32) / 10.0 + (temp_dec as f32);

                let device_data = MeterProCo2Data {
                    device_type: "MeterPro(CO2)".to_string(),
                    device_mac: properties.address.to_string(),
                    temperature,
                    scale: "CELSIUS".to_string(),
                    humidity,
                    co2,
                    battery,
                    time_of_sample: SystemTime::now()
                        .duration_since(UNIX_EPOCH)
                        .unwrap()
                        .as_millis(),
                };

                return Ok(Some(device_data));
            }
        }
        Ok(None)
    }
}
