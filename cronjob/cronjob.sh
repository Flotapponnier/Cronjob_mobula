#!/bin/bash

# Entry script to start cron service and keep container alive

echo "Starting snapshot container..."
echo "$(date): Container started" >> /var/log/cron.log

# Start cron service
service cron start

echo "Cron service started - Snapshots scheduled every 5 minutes"
echo "$(date): Cron service started" >> /var/log/cron.log

# Display active cron configuration
echo "Active cron configuration:"
crontab -l

# Keep container alive and display logs
echo "Monitoring cron logs (Ctrl+C to stop):"
tail -f /var/log/cron.log