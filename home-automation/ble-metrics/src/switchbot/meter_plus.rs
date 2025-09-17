use crate::switchbot::switchbot_device_scanner::{SwitchBotDeviceScanner, SERVICE_UUID_STR};
use btleplug::api::PeripheralProperties;
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};
use uuid::Uuid;

#[derive(Debug, Serialize, Deserialize)]
pub struct MeterPlusData {
    pub device_type: String,  //	the type of the device
    pub device_mac: String,   //	the MAC address of the device
    pub temperature: f32,     //	the current temperature reading
    pub scale: String,        //	the current temperature unit being used
    pub humidity: u8,         //	the current humidity reading in percentage
    pub battery: u8,          //	the current battery level, 0-100
    pub time_of_sample: u128, //	the time stamp when the event is sent
}

pub struct MeterPlusScanner {}

impl SwitchBotDeviceScanner<MeterPlusData> for MeterPlusScanner {
    fn scan(
        properties: &PeripheralProperties,
    ) -> Result<Option<MeterPlusData>, Box<dyn std::error::Error>> {
        let target_service_uuid = Uuid::parse_str(SERVICE_UUID_STR)?;
        if let Some(service_data) = properties.service_data.get(&target_service_uuid) {
            let battery = service_data[2] & 0b01111111;

            if let Some(manufacturer_data) = properties.manufacturer_data.values().next() {
                if manufacturer_data.len() != 11 {
                    return Ok(None);
                }

                let humidity = service_data[5] & 0b01111111;
                let temp_int = service_data[3] & 0b00001111;
                let temp_dec = service_data[4] & 0b01111111;
                let temperature = (temp_int as f32) / 10.0 + (temp_dec as f32);

                let device_data = MeterPlusData {
                    device_type: "WoMeter".to_string(),
                    device_mac: properties.address.to_string(),
                    temperature,
                    scale: "CELSIUS".to_string(),
                    humidity,
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
