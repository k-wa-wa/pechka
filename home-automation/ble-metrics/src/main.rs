mod switchbot;

use crate::switchbot::meter_plus::MeterPlusScanner;
use crate::switchbot::switchbot_device_scanner::SwitchBotDeviceScanner;
use btleplug::api::{Central, Manager as _, Peripheral, ScanFilter};
use btleplug::platform::Manager;
use std::collections::HashSet;
use std::env;
use switchbot::meter_pro_co2_scanner::MeterProCo2Scanner;
use tokio::time::{sleep, Duration};

struct Config {
    scan_interval_secs: u64,
    enable_meter_pro_co2_scanner: bool,
    enable_meter_plus_scanner: bool,
}

impl Config {
    fn new() -> Self {
        let scan_interval_secs = env::var("SWITCHBOT_SCAN_INTERVAL_SECS")
            .unwrap_or_else(|_| "1".to_string())
            .parse::<u64>()
            .unwrap_or(1);

        let target_devices: HashSet<String> = env::var("SWITCHBOT_TARGET_DEVICES")
            .unwrap_or_else(|_| "".to_string())
            .split(',')
            .filter(|s| !s.trim().is_empty())
            .map(|s| s.trim().to_lowercase())
            .collect();

        let enable_meter_pro_co2_scanner =
            target_devices.is_empty() || target_devices.contains("meterproco2scanner");
        let enable_meter_plus_scanner =
            target_devices.is_empty() || target_devices.contains("meterplus");

        Config {
            scan_interval_secs,
            enable_meter_pro_co2_scanner,
            enable_meter_plus_scanner,
        }
    }
}

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

    let config = Config::new();

    loop {
        let devices = central.peripherals().await?;
        for device in devices {
            if let Ok(Some(properties)) = device.properties().await {
                if config.enable_meter_pro_co2_scanner {
                    if let Some(data) = MeterProCo2Scanner::scan(&properties)? {
                        println!("{}", serde_json::to_string(&data)?);
                    }
                }
                if config.enable_meter_plus_scanner {
                    if let Some(data) = MeterPlusScanner::scan(&properties)? {
                        println!("{}", serde_json::to_string(&data)?);
                    }
                }
            }
        }
        sleep(Duration::from_secs(config.scan_interval_secs)).await;
    }
}
