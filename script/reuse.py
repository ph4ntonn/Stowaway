#!/usr/bin/env python
# -*- coding: utf-8 -*-

import socket
import argparse
import sys
import hashlib

# eg: when start node like: ./linux_x64_agent --report 80 -l 10000 -s ph4ntom ,set SECRET = "ph4ntom"
SECRET = "" # set SECRET as you secret key(-s option) ,if you do not set the -s option,just leave it like SECRET = ""

# Usage:
# python reuse.py --start --rhost 192.168.1.2 --rport 80  Start the port reuse function
# python reuse.py --stop --rhost 192.168.1.2 --rport 80 Stop the port reuse function

parser = argparse.ArgumentParser(description='start/stop iptables port reuse')
parser.add_argument('--start', help='start port reusing', action='store_true')
parser.add_argument('--stop', help='stop port reusing', action='store_true')
parser.add_argument('--rhost', help='remote host', dest='ip')
parser.add_argument('--rport', help='remote port', dest='port')

first_checkcode = hashlib.md5(SECRET.encode()).hexdigest()
second_checkcode = hashlib.md5(first_checkcode.encode()).hexdigest()
final_checkcode = first_checkcode[:24] + second_checkcode[:24]

START_PORT_REUSE = final_checkcode[16:32]
STOP_PORT_REUSE = final_checkcode[32:]

options = parser.parse_args()    

data = ""

if options.start:
    data = START_PORT_REUSE
elif options.stop:
    data = STOP_PORT_REUSE
else:
    parser.print_help()
    sys.exit(0)

try:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.settimeout(2)
    s.connect((options.ip, int(options.port)))
    s.send(data.encode())
except:
    print("[*] Cannot connect to target")

try:
    s.recv(1024)
except:
    pass

s.close()

print("[*] Done!")