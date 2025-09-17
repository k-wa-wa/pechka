use btleplug::api::PeripheralProperties;
use serde::{Serialize};

pub const SERVICE_UUID_STR: &str = "0000fd3d-0000-1000-8000-00805f9b34fb";

pub trait SwitchBotDeviceScanner<T>
where
    T: Serialize,
{
    fn scan(properties: &PeripheralProperties) -> Result<Option<T>, Box<dyn std::error::Error>>;
}
