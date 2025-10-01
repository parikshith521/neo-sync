# neo-sync
A P2P file synchronization tool written in Go. Keeps directories in sync across multiple locations, eventually using a direct peer-to-peer network without a central server.

## progress
The core synchronization engine is complete. Currently we can watch and sync two local directories.

I'm currently working on the two-peer networking part which involves writing a TCP-based protocol for exchanging file states and requested file data. I also plan to implement automatic peer discovery using mDNS.