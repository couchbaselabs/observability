const child_process = require("child_process");
const fs = require("fs");
const path = require("path");
const puppeteer = require("puppeteer");

function getOptions() {
    const dashboardsBase = path.resolve(".", "../../microlith/grafana/provisioning/dashboards");
    try {
        if ("GITHUB_ACTIONS" in process.env) {
            let files;
            const context = require("@actions/github").context;
            if ("inputs" in context.payload) {
                if (context.payload.inputs.files === "all") {
                    files = fs.readdirSync(dashboardsBase).map(x => path.join(dashboardsBase, x)).filter(x => /\.json$/.test(x));
                } else {
                    files = context.payload.inputs.files.split(/,\s?/).map(x => path.resolve(dashboardsBase, x + ".json"));
                }
                let pullRequest;
                if (context.payload.inputs.pr) {
                    pullRequest = parseInt(context.payload.inputs.pr);
                }
                return {
                    files,
                    pullRequest
                };
            } else {
                const base = context.payload.pull_request.base.sha;
                const head = context.payload.pull_request.head.sha;
                // Ignore deleted files
                const changedFiles = child_process.execSync(`git diff --diff-filter=d --name-only ${base} ${head} `, { encoding: "ascii" }).trim().split("\n");
                return {
                    files: changedFiles,
                    pullRequest: context.payload.pull_request.number
                };
            }
        } else {
            if (process.argv.length < 3) {
                throw new Error("Insufficient arguments. Usage: node index.js [comma-separated-files | 'all']");
            }
            const filesInput = process.argv[2];
            if (filesInput.trim() === "all") {
                files = fs.readdirSync(dashboardsBase).map(x => path.join(dashboardsBase, x)).filter(x => /\.json$/.test(x));
            } else {
                files = filesInput.split(/,\s?/).map(x => path.resolve(dashboardsBase, x + ".json"));
            }
            return { files, pullRequest: undefined };
        }
    } catch(e) {
        console.error("Error detected so defaulting to all dashboards", e);
        files = fs.readdirSync(dashboardsBase).map(x => path.join(dashboardsBase, x)).filter(x => /\.json$/.test(x));
        return { files, pullRequest: undefined };
    }
}

(async function(){
    process.argv.forEach((val, index) => {
        console.log(`${index}: ${val}`);
      });

    const {files, pullRequest} = getOptions();
    console.log("Taking screenshots of", files);

    const browser = await puppeteer.launch({ headless: true });
    const screenshots = [];

    var additionalQueryArgs = process.env.GRAFANA_ADDITIONAL_QUERY_ARGS
    if ( additionalQueryArgs.length > 0 && !additionalQueryArgs.startsWith(`?`) ) {
        additionalQueryArgs = `?` + additionalQueryArgs
    }

    await Promise.all(files.map(async file => {
        const absolute = path.resolve(__dirname, "../..", file);
        const base = path.basename(file).replace(".json", "");
        const definition = require(absolute);
        const uid = definition.uid;

        if (uid == null) {
            console.error("Missing UID for", absolute)
            return
        }

        // Recommended resolution is 1920x1080
        const recommendedPage = await browser.newPage();
        await recommendedPage.setViewport({ width: 1920, height: 1080 });
        // Minimum resolution is 1366x768
        const minimumSizePage = await browser.newPage();
        await minimumSizePage.setViewport({ width: 1366, height: 768 });

        const grafanaUrl = `http://localhost:8080/grafana/d/${uid}/${base}${additionalQueryArgs}`
        console.log("Screenshotting", grafanaUrl)

        await recommendedPage.goto(grafanaUrl);
        await recommendedPage.waitForNetworkIdle({timeout: 0, idleTime:5000});
        const recommendedScreenshotPath = path.join(__dirname, `${base}-1920x1080.png`);
        await recommendedPage.screenshot({ path: recommendedScreenshotPath });
        screenshots.push(recommendedScreenshotPath);

        await minimumSizePage.goto(grafanaUrl);
        await minimumSizePage.waitForNetworkIdle({timeout: 0, idleTime:5000});
        const minimumScreenshotPath = path.join(__dirname, `${base}-1366x768.png`);
        await minimumSizePage.screenshot({ path: minimumScreenshotPath });
        screenshots.push(minimumScreenshotPath);
    }));

    await browser.close();

    console.log("Screenshots saved: " + screenshots);
})();
