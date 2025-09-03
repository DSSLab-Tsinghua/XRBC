# Efficient Asynchronous Cross-Consensus Reliable Broadcast


[![Conference](https://img.shields.io/badge/Conference-NDSS%20'26-blue)](https://www.ndss-symposium.org/ndss2026/)
[![Data](https://img.shields.io/badge/Data-Zenodo-4c7e9b.svg)](https://zenodo.org/records/16945739)
[![DOI](https://zenodo.org/badge/DOI/10.5281/zenodo.16945739.svg)](https://doi.org/10.5281/zenodo.16945739)


This repository contains the **code, and instructions** for reproducing the experiments and results from our paper:

> **Cross-Consensus Reliable Broadcast and its Applications**  
> Yue Huang, Xin Wang, Haibin Zhang, Sisi Duan  
> Network and Distributed System Security (NDSS) Symposium 2026  
> [📄 Paper PDF](https://eprint.iacr.org/2025/1483)

---



## 🔬 Introduction

We introduce **Cross-Consensus Reliable Broadcast (XRBC)** - a novel primitive that models secure communication between distributed groups where at least one group executes a consensus protocol. Our experimental evaluation demonstrates that XRBC protocols achieve:

- ⚡ **Low latency** communication across consensus boundaries
- 📈 **High throughput** under various network conditions  
- 🛡️ **Strong security** guarantees for cross-group interactions

### 🧪 Artifact Overview
This artifact enables reproduction of **Section 7** results from our NDSS 2026 paper. The experiments require no specialized hardware and run on standard commodity machines.

**Test Environment:**
- 🖥️ 4-core CPU, 16GB RAM, 100GB storage
- 🌐 100 Mbps network connection  
- 🐧 Linux operating system

> **Note**: While our paper presents full-scale results on Amazon EC2, this artifact provides scaled-down experiments for single-machine reproduction. For EC2 deployment instructions, see the AWS experiment overview below.

## 🚀 How to Run the Experiments

### Prerequisites
```bash
# Install dependencies
./autoEnv.bash
```

### Experiments

#### 🔬 **Experiment 1** - Claim #1
```bash
./autoE1.bash
```
📄 Results will be saved to `resultE1.txt`

#### 🔬 **Experiment 2** - Claim #2  
```bash
./autoE2.bash
```
📄 Results will be saved to `resultE2.txt`  
⏱️ *This experiment may take longer - please wait for completion*

#### 🔬 **Experiment 3** - Claim #3
```bash
./autoE3.bash
```
📄 Results will be saved to `resultE3.txt`  
⏱️ *This experiment may take longer - please wait for completion*

## 🛠️ Script Descriptions

### Environment Setup
```bash
./autoEnv.bash    # Install all dependencies
```

### Configuration
```bash
./autoConfig.bash [m] [n] [Consensus No] etc/conf.json
```
Modify the configuration file in `etc/conf.json`

### Experiment Scripts
| Script | Purpose |
|--------|---------|
| `./autoE1.bash` | Run Experiment 1 (Claim #1) |
| `./autoE2.bash` | Run Experiment 2 (Claim #2) |
| `./autoE3.bash` | Run Experiment 3 (Claim #3) |

## ⚠️ Important Notes

### 💻 Local Testing
> **IDE Users**: Running `autoE1.bash`, `autoE2.bash`, and `autoE3.bash` may freeze your IDE due to intensive resource usage. After starting a script, wait ~2 minutes. If the IDE becomes unresponsive, close it and restart the experiments. Progress logs are saved in `resultE1.txt`, `resultE2.txt`, and `resultE3.txt`.

> **OS Compatibility**: We recommend **Ubuntu**. Other operating systems (e.g., CentOS) may cause dependency downloads (like `jq`) to fail or timeout.

### 📊 Hardware Variability (E3)
When running all logical XRBC nodes on a single machine, they compete for shared resources (CPU time, cache/memory bandwidth, kernel networking). Once hardware resources are saturated, each additional node adds only limited, stable latency overhead - total latency grows predictably without superlinear escalation.

Different machines have varying available headroom before reaching partial saturation, so reported latencies may vary widely between systems. This variability is **expected** and doesn't contradict our claims of controlled, predictable growth with bounded per-node incremental overhead (see `resultE3.txt`).

## ☁️ AWS Experiment (Overview)

For large-scale testing on Amazon EC2:

1. **🚀 Launch**  
   Spin up Ubuntu 22.04 VMs in four regions with boto2; confirm they are running.

2. **📦 Deploy**  
   Install Go 1.21 and dependencies, then push binaries and configs to every node via fab2.

3. **▶️ Run**  
   Start servers and clients remotely with fab2, then pull back the log files.

4. **📊 Analyze**  
   Parse the collected logs to compute latency and throughput.

> **Note**: This repository focuses on local testing. EC2-related deployment codes for AWS testing are not included in this release.

## 📖 Citation

If you find this work useful, please cite our paper:

```bibtex
@inproceedings{huang2026xrbc,
  title={Cross-Consensus Reliable Broadcast and its Applications},
  author={Huang, Yue and Wang, Xin and Zhang, Haibin and Duan, Sisi},
  booktitle={Proceedings of the 2026 Network and Distributed System Security Symposium},
  year={2026},
  organization={Internet Society}
}
```