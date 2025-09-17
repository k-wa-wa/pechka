use crate::switchbot::switchbot_device_scanner::{SwitchBotDeviceScanner, SERVICE_UUID_STR};
use btleplug::api::PeripheralProperties;
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};
use uuid::Uuid;

#[derive(Debug, Serialize, Deserialize)]
pub struct ContactSensorData {
    pub device_type: String,     //	the type of the device
    pub device_mac: String,      //	the MAC address of the device
    pub detection_state: String, //	the motion state of the device, "DETECTED" stands for motion is detected; "NOT_DETECTED" stands for motion has not been detected for some time
    pub door_mode: String, //	when the enter or exit mode gets triggered, "IN_DOOR" or "OUT_DOOR" is returned
    pub brightness: String, //	the level of brightness, can be "bright" or "dim"
    pub open_state: String, //	the state of the contact sensor, can be "open" or "close" or "timeOutNotClose"
    pub battery: u8,        //	the current battery level, 0-100
    pub time_of_sample: u128, //	the time stamp when the event is sent
}

pub struct ContactSensorScanner {}

impl SwitchBotDeviceScanner<ContactSensorData> for ContactSensorScanner {
    fn scan(
        properties: &PeripheralProperties,
    ) -> Result<Option<ContactSensorData>, Box<dyn std::error::Error>> {
        let target_service_uuid = Uuid::parse_str(SERVICE_UUID_STR)?;
        if let Some(service_data) = properties.service_data.get(&target_service_uuid) {
            let battery = service_data[2] & 0b01111111;

            if let Some(manufacturer_data) = properties.manufacturer_data.values().next() {
                if manufacturer_data.len() != 13 {
                    return Ok(None);
                }

                let light_level = service_data[3] & 1;
                let brightness = match light_level {
                    0 => "dim".to_string(),
                    1 => "bright".to_string(),
                    _ => "unknown".to_string(),
                };

                let hal_state: u8 = (service_data[3] >> 1) & 0b11;
                let open_state = match hal_state {
                    0 => "close".to_string(),
                    1 => "open".to_string(),
                    2 => "timeOutNotClose".to_string(),
                    _ => "unknown".to_string(), // Handle unexpected values
                };

                let device_data = ContactSensorData {
                    device_type: "WoContact".to_string(),
                    device_mac: properties.address.to_string(),
                    detection_state: "not_implemented".to_string(),
                    door_mode: "not_implemented".to_string(),
                    brightness,
                    open_state,
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
