#!/usr/bin/env python3
# Socket client for the collector socket input plugin

import socket
import time
import sys
import argparse
import json

def parse_args():
    parser = argparse.ArgumentParser(description='Socket client for the collector')
    parser.add_argument('--host', default='localhost', help='Server hostname')
    parser.add_argument('--port', type=int, default=8888, help='Server port')
    parser.add_argument('--interval', type=float, default=1.0, help='Interval between messages (seconds)')
    parser.add_argument('--format', choices=['text', 'json'], default='text', help='Message format')
    parser.add_argument('message', nargs='?', default='Test message from socket client', help='Message to send')
    return parser.parse_args()

def main():
    args = parse_args()
    
    # Create a socket connection
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    
    try:
        # Connect to the server
        sock.connect((args.host, args.port))
        print(f"Connected to {args.host}:{args.port}")
        
        count = 0
        try:
            while True:
                count += 1
                timestamp = time.strftime("%Y-%m-%d %H:%M:%S", time.localtime())
                
                # Create the message
                if args.format == 'json':
                    message = json.dumps({
                        "timestamp": timestamp,
                        "message": f"{args.message} - #{count}",
                        "level": "INFO"
                    }) + "\n"
                else:
                    message = f"{timestamp} - {args.message} - #{count}\n"
                
                # Send the message
                sock.sendall(message.encode('utf-8'))
                print(f"Sent: {message.strip()}")
                
                # Wait for the specified interval
                time.sleep(args.interval)
                
        except KeyboardInterrupt:
            print("\nStopping...")
    
    except ConnectionRefusedError:
        print(f"Connection to {args.host}:{args.port} refused. Is the server running?")
        return 1
    finally:
        sock.close()
    
    return 0

if __name__ == "__main__":
    sys.exit(main())