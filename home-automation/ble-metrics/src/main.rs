mod switchbot;

use btleplug::api::{Central, Manager as _, Peripheral, ScanFilter};
use btleplug::platform::Manager;
use std::env;
use switchbot::meter_pro_co2_scanner::MeterProCo2Scanner;
use crate::switchbot::meter_plus::MeterPlusScanner;
use crate::switchbot::switchbot_device_scanner::SwitchBotDeviceScanner;
use tokio::time::{sleep, Duration};
#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let manager = Manager::new().await?;
    let central = manager
        .adapters()
        .await?
        .into_iter()
        .next()
        .ok_or("No Bluetooth adapter found")?;

    central.start_scan(ScanFilter::default()).await?;

    loop {
        let devices = central.peripherals().await?;
        for device in devices {
            if let Ok(Some(properties)) = device.properties().await {
                if let Some(data) = MeterProCo2Scanner::scan(&properties)? {
                    println!("{}", serde_json::to_string(&data)?);
                }
                if let Some(data) = MeterPlusScanner::scan(&properties)? {
                    println!("{}", serde_json::to_string(&data)?);
                }
            }
        }
        let sleep_duration_secs = env::var("SWITCHBOT_SCAN_INTERVAL_SECS")
            .unwrap_or_else(|_| "1".to_string())
            .parse::<u64>()
            .unwrap_or(1);
        sleep(Duration::from_secs(sleep_duration_secs)).await;
    }
}
