import { app, BrowserWindow, ipcMain, dialog } from 'electron';
import path, { resolve } from 'node:path';
import started from 'electron-squirrel-startup';
import { spawn } from 'node:child_process';
import { rejects } from 'node:assert';
// Handle creating/removing shortcuts on Windows when installing/uninstalling.
if (started) {
  app.quit();
}

const createWindow = () => {
  // Create the browser window.
  const mainWindow = new BrowserWindow({
    width: 800,
    height: 600,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: false,
      nodeIntegration: true,
    },
  });

  // and load the index.html of the app.
  if (MAIN_WINDOW_VITE_DEV_SERVER_URL) {
    mainWindow.loadURL(MAIN_WINDOW_VITE_DEV_SERVER_URL);
  } else {
    mainWindow.loadFile(path.join(__dirname, `../renderer/${MAIN_WINDOW_VITE_NAME}/index.html`));
  }

  // Open the DevTools.
  mainWindow.webContents.openDevTools();
};

// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.
app.whenReady().then(() => {
  createWindow();

  // On OS X it's common to re-create a window in the app when the
  // dock icon is clicked and there are no other windows open.
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

// Quit when all windows are closed, except on macOS. There, it's common
// for applications and their menu bar to stay active until the user quits
// explicitly with Cmd + Q.
app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

let goProcess = null;


ipcMain.handle('select-folder', async ()=>{
  const result = await dialog.showOpenDialog({
    properties: ['openDirectory']
  })
  if(result.canceled) return null;
  return result.filePaths[0];
})

ipcMain.handle('start-go-server', (event, folder, port, username) => {
  if (goProcess) return 'Server already running';

  const gopath = resolve(process.cwd(), 'src/server/server.exe');  

  console.log("Starting Go server at:", gopath, folder, port, username);

  goProcess = spawn(gopath, [folder, port, username], {
    cwd: resolve(process.cwd(), 'src/server'), 
  });

  goProcess.stdout.on('data', (data) => console.log(`GO STDOUT: ${data.toString()}`));
  goProcess.stderr.on('data', (data) => console.error(`GO STDERR: ${data.toString()}`));
  goProcess.on('error', (err) => console.error("Failed to start Go server:", err));
  goProcess.on('close', (code) => {
    console.log("Go server exited with code:", code);
    goProcess = null;
  });

  return 'Go server process started';
});

let cfProcess = null;

ipcMain.handle('start-tunnel', async(event, port) =>{
  if(cfProcess) return 'Already exposed'
  return new Promise((resolve, reject) =>{
    cfProcess = spawn('cloudflared', ['tunnel', '--url', `http://localhost:${port}`]);
     const timeout = setTimeout(() => {
      reject(new Error('Failed to get public URL from Cloudflare Tunnel'));
    }, 10000);
    cfProcess.stderr.on('data', (data)=>{
      const lines = data.toString().split(/\r?\n/);
      lines.forEach(line => {
        const cleaned = line.replace(/[| ]+/g, '');
        const match = cleaned.match(/https:\/\/[a-z0-9\-]+\.trycloudflare\.com/);
        if (match) {
          console.log("Public URL found:", match[0]);
          resolve(match[0]);
        }});
    })        
    cfProcess.on('close', (code) => {
        console.log('Tunnel process exited with code', code);
        cfProcess = null;
      });
  })

})