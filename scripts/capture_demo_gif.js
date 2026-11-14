const puppeteer = require('B:/workgit/node_modules/puppeteer');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const framesDir = 'B:/workgit/ragfuzz/temp_gif_frames';
if (!fs.existsSync(framesDir)) {
  fs.mkdirSync(framesDir, { recursive: true });
} else {
  // Clean old frames
  fs.readdirSync(framesDir).forEach(f => fs.unlinkSync(path.join(framesDir, f)));
}

async function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

let frameIndex = 0;
async function captureFrames(page, count, delayMs = 100) {
  for (let i = 0; i < count; i++) {
    const filename = path.join(framesDir, `frame_${String(frameIndex).padStart(4, '0')}.png`);
    await page.screenshot({ path: filename });
    frameIndex++;
    if (delayMs > 0) await sleep(delayMs);
  }
}

(async () => {
  console.log('[+] Starting Puppeteer browser for GIF recording...');
  const browser = await puppeteer.launch({
    headless: 'new',
    executablePath: 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1280,800']
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1280, height: 800, deviceScaleFactor: 1 });

  await page.goto('http://localhost:4040', { waitUntil: 'domcontentloaded' });
  await sleep(1500);

  // Phase 1: Dashboard overview
  console.log('[+] Recording Phase 1: Dashboard Overview...');
  await captureFrames(page, 14, 100);

  // Phase 2: Click Query Inspector
  console.log('[+] Recording Phase 2: Query Inspector...');
  await page.evaluate(() => {
    const btn = Array.from(document.querySelectorAll('button')).find(b => b.textContent.includes('Query Inspector'));
    if (btn) btn.click();
  });
  await sleep(300);
  await captureFrames(page, 10, 100);

  // Phase 3: Expand test case TC-DST-0001
  console.log('[+] Recording Phase 3: Expand Test Case...');
  await page.evaluate(() => {
    const target = Array.from(document.querySelectorAll('div, button')).find(el => el.textContent && el.textContent.includes('TC-DST-0001'));
    if (target) target.click();
  });
  await sleep(400);
  await captureFrames(page, 16, 100);

  // Phase 4: Click Topology Graph
  console.log('[+] Recording Phase 4: Topology Graph...');
  await page.evaluate(() => {
    const btn = Array.from(document.querySelectorAll('button')).find(b => b.textContent.includes('Topology Graph'));
    if (btn) btn.click();
  });
  await sleep(400);
  await captureFrames(page, 12, 100);

  // Phase 5: Select a node in the graph
  console.log('[+] Recording Phase 5: Select Graph Node...');
  await page.evaluate(() => {
    const circles = Array.from(document.querySelectorAll('circle'));
    if (circles.length > 2) circles[2].parentElement.dispatchEvent(new MouseEvent('click', { bubbles: true }));
  });
  await sleep(400);
  await captureFrames(page, 14, 100);

  // Phase 6: Click Live Playground
  console.log('[+] Recording Phase 6: Live Playground...');
  await page.evaluate(() => {
    const btn = Array.from(document.querySelectorAll('button')).find(b => b.textContent.includes('Live Playground'));
    if (btn) btn.click();
  });
  await sleep(400);
  await captureFrames(page, 10, 100);

  // Phase 7: Run Adversarial Probe
  console.log('[+] Recording Phase 7: Run Adversarial Probe...');
  await page.evaluate(() => {
    const btn = Array.from(document.querySelectorAll('button')).find(b => b.textContent.includes('Run Adversarial Probe'));
    if (btn) btn.click();
  });
  await sleep(1500); // Allow probe to finish
  await captureFrames(page, 18, 100);

  await browser.close();
  console.log(`[+] Captured ${frameIndex} frames. Rendering GIF with ffmpeg...`);

  const gifOutput = 'B:/workgit/ragfuzz/docs/images/ragfuzz_demo.gif';
  const ffmpegCmd = `ffmpeg -y -framerate 10 -i "${framesDir}/frame_%04d.png" -vf "fps=10,scale=1000:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128[p];[s1][p]paletteuse=dither=bayer:bayer_scale=3" "${gifOutput}"`;

  execSync(ffmpegCmd, { stdio: 'inherit' });
  console.log(`[+] GIF successfully created at ${gifOutput}!`);

  // Clean up temporary frames
  fs.readdirSync(framesDir).forEach(f => fs.unlinkSync(path.join(framesDir, f)));
  fs.rmdirSync(framesDir);
  console.log('[+] Temporary frame directory cleaned up.');
})();
