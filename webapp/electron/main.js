const electron = require('electron');
const app = electron.app;
const Menu = electron.Menu;
const BrowserWindow = electron.BrowserWindow;
const { spawn } = require("child_process");

function buildMenu() {
    const template = [
        {
            label: "Maker",
            submenu: [
                {role: "quit"},
                {role: "about"},
            ]
        },
        {
            label: 'View',
            submenu: [
                {role: 'reload'},
                {role: 'forcereload'},
                {role: 'toggledevtools'},
                {type: 'separator'},
                {role: 'resetzoom'},
                {role: 'zoomin'},
                {role: 'zoomout'},
                {type: 'separator'},
                {role: 'togglefullscreen'}
            ]
        },
        {
            role: 'window',
            submenu: [
                {role: 'minimize'},
                {role: 'close'}
            ]
        },
    ];

    if (process.platform === 'darwin') {
        template.unshift({
            label: app.getName(),
            submenu: [
                {role: 'hide'},
                {role: 'hideothers'},
                {role: 'unhide'},
                {type: 'separator'},
                {role: 'quit'}
            ]
        });

        // Window menu
        template[3].submenu = [
            {role: 'close'},
            {role: 'minimize'},
            {role: 'zoom'},
            {type: 'separator'},
            {role: 'front'}
        ]
    }

    const menu = Menu.buildFromTemplate(template);
    Menu.setApplicationMenu(menu);
}

function createWindow() {
    window = new BrowserWindow({
        width: 1280,
        height: 800,
        webPreferences: {
            webSecurity: false,
        },
        show: false,
    });

    //window.openDevTools();

    //mainWindow.loadURL(`file://${__dirname}/app/index.html`)
    let r = window.loadURL("http://localhost:6045/index.html");
    console.log(r);

    window.once("ready-to-show", () => {
        console.log("window is ready to show");
        window.show();
    });

    // Emitted when the window is closed.
    window.on('closed', function (e) {
        window = null;
    });

    buildMenu();
}

let serverIsReady = false;
const server = spawn('./maker', ['server']);
server.stderr.on("data", (data) => {
    const buf = `${data}`;
    if (buf.indexOf("Starting server")) {
        serverIsReady = true;
    }
    process.stderr.write(data);
});

app.on('ready', () => {
    const interval = setInterval(() => {
        if (serverIsReady) {
            clearInterval(interval);
            setTimeout(() => {
                createWindow();
            }, 100);
        }
    }, 100);
});

// Quit when all windows are closed.
app.on('window-all-closed', function () {
    // On OS X it is common for applications and their menu bar
    // to stay active until the user quits explicitly with Cmd + Q
    if (process.platform !== 'darwin') {
        app.quit();
    }
});

app.on('activate', function () {
    // On OS X it's common to re-create a window in the app when the
    // dock icon is clicked and there are no other windows open.
    if (window === null) {
        createWindow();
    }
});

app.on("quit", () => {
    server.kill("SIGTERM");
});