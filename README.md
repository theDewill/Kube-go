
## 📁 Project - Kube

**Kube** is a locally deployable smart storage solution featuring:

* 🔍 Intelligent file search
* 🧊 Advanced compression modules (called **Refrigerators**)
* 🔐 Persistent intelligent security protocols

---

## 🚀 Quick Start (Recommended)
[Run below 2 commands from project root]
1) Run the darwin_deps_install.sh to setup necessary folders and file movements befor running the application

```bash
sh darwin_deps_install.sh
```

2) Run Kube directly via the prebuilt executable (macOS ARM M-Series only)

```bash
./kube-go
```

> ⚠️ Note: This binary is prebuilt for **macOS ARM (M Sereies)** systems.

---

## 🛠️ Installation & Build Guide

### ⚙️ Supported Platforms

* ✅ macOS (ARM/Intel)
* ✅ Linux
* ⚠️ Windows (Supported, but setup is complex)

---

### 🐧 macOS / Linux

#### 1. Install Prerequisites:

* [Golang 1.24.3+](https://go.dev/dl/)
* [GoCV & OpenCV]
  [→ macOS Guide](https://gocv.io/getting-started/macos/)
  [→ Linux Guide](https://gocv.io/getting-started/linux/)

#### 2. Set Up Project:

From the root of the project:

```bash
# For macOS
sh darwin_deps_install.sh

# For Linux
sh linux_deps_install.sh
```

#### 3. Run or Build the App:

* Run in development mode:

  ```bash
  wails dev
  ```
* Build the application:

  ```bash
  wails build
  ```

  Then navigate to the output directory and run the executable.

---

### 🪟 Windows (Advanced Users)

> ⚠️ The installation process on Windows is **significantly more complex** due to required system dependencies like CMake and MinGW-W64.

#### Resources:

* [GoCV Windows Guide](https://gocv.io/getting-started/windows/)
* [Third-party Guide (Indonesian)](https://kekasi.co.id/cara-install-gocv-dan-opencv-di-windows-10/)
* [Download Dependencies (Google Drive)](https://drive.google.com/drive/folders/1bNWxMUP0oztWcGGWtNx4yPb2FiSeQwuP?usp=sharing)

---

## 📌 Notes

* This project is designed and tested primarily for **macOS and Linux** platforms.
* Windows support is **not officially maintained** as part of this submission.
* Demo video and screenshots included separately to showcase functionality.

---

## 📫 Author

Feel free to reach out with any questions or feedback.
🧑‍💻 Your Name – `your.email@domain.com`

---

### ✅ Final Tips:

* Replace `YOUR_LINK_HERE` with the actual Google Drive link.
* You can also add a section with **screenshots** or **GIFs** showing the app in action if space permits.

Let me know if you'd like this saved as a downloadable file or embedded into your project structure.
