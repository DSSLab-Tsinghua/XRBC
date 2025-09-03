# Efficient Asynchronous Cross-Consensus Reliable Broadcast


[![Conference](https://img.shields.io/badge/Conference-NDSS%20'25-blue)](https://www.ndss-symposium.org/ndss2026/)
[![Data](https://img.shields.io/badge/Data-Zenodo-4c7e9b.svg)](https://zenodo.org/records/16945739)
[![DOI](https://zenodo.org/badge/DOI/10.5281/zenodo.16945739.svg)](https://doi.org/10.5281/zenodo.16945739)


This repository contains the **code, and instructions** for reproducing the experiments and results from our paper:

> **Cross-Consensus Reliable Broadcast and its Applications**  
> Yue Huang, Xin Wang, Haibin Zhang, Sisi Duan  
> Network and Distributed System Security (NDSS) Symposium 2026  
> [📄 Paper PDF](https://eprint.iacr.org/2025/1483)

---



## Introduction
We propose a new primitive called cross-consensus reliable broadcast (XRBC). The XRBC primitive models the security properties of communication between two groups, where at least one group executes a consensus protocol. Our experimental results show that all of our protocols achieve low latency and decent throughput.

This artifact demonstrates how to reproduce the results presented in Section 7 of our paper. The experiments do not require any specialized hardware. Our test environment is a computer equipped with a 4-core CPU, 16 GB of RAM, 100 GB of storage, a 100 Mbps network connection, and the Linux operating system.

Note that all results reported in our paper require access to the Amazon EC2. In this artifact appendix, we present the workflow to reproduce scaled-down experiments using one machine. For readers who are familiar with reproducing the results on EC2, please check the overview instructions.

## How to Run the experiments
1. Download the repository from the Zenodo link: https://zenodo.org/records/16945739, unzip it, and navigate to the repo directory (as $Home):
```bash
$ unzip xrbc-ae-25.zip -d xrbc-ae-25
$ cd xrbc-ae-25
```

2. Download all dependencies
```bash
$ ./autoEnv.bash
```

3. Start the first experiment for Claim#1, and you can check the result at resultE1.txt ($Home)
```bash
$ ./autoE1.bash
```

4. Start the second experiment for Claim#2, and you can check the result at resultE2.txt ($Home)
```bash
$ ./autoE2.bash
```
（This experiment may cost longer waiting time, please wait for the completion）

5. Start the second experiment for Claim#2, and you can check the result at resultE3.txt ($Home)
```bash
$ ./autoE3.bash
```
（This experiment may cost longer waiting time, please wait for the completion）

## Description of scripts

1. Download all dependencies
```bash
$ ./autoEnv.bash
```

2. Modify the configuration file in the directory: $Home:/etc/conf.json
```bash
$ ./autoConfig.bash [m] [n] [Consensus No] etc/conf.json
```

3. Experiment scripts
```bash
$ ./autoE1.bash
$ ./autoE2.bash
$ ./autoE3.bash
```

## Local test note
If one uses IDE to run the scripts, executing autoE1.bash, autoE2.bash, and autoE3.bash may freeze the IDE due to the large computer resources taken to launch the experiments. After starting the bash script, wait for about two minutes. If the IDE is not responsible, please close the IDE and restart the experiments. As mentioned above, the data log in resultE1.txt, resultE2.txt, resultE3.txt is updated if the experiments are successfully launched. 

We recommend Ubuntu; other operating systems (e.g., CentOS) may cause dependency (e.g., jq) downloads to fail or time out.

## Local Hardware Variability Note (related to E3)

When all logical XRBC nodes are executed on a single local machine, they compete for shared CPU time, cache/memory bandwidth, and kernel networking (loopback queues). Once the resources of the hardware is saturated, each additional node adds only a limited and stable amount of extra latency; total latency therefore grows without superlinear escalation. Because different machines have different available headroom before reaching this partial saturation point, latencies reported by different machines may vary widely. This variability is expected and does not contradict the claim of controlled, predictable growth with bounded per‑node incremental overhead (see resultE3.txt).

## AWS experiment (Overview)
1. Launch
Spin up Ubuntu 22.04 VMs in four regions with boto2; confirm they are running.
2. Deploy
Install go 1.21 and dependencies, then push binaries and configs to every node via fab2.
3. Run
Start servers and clients remotely with fab2, then pull back the log files.
4. Analyze
Parse the collected logs to compute latency and throughput.


