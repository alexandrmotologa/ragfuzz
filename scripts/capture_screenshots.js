const puppeteer = require('B:/workgit/node_modules/puppeteer');
const path = require('path');
const fs = require('fs');

async function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

(async () => {
  const browser = await puppeteer.launch({
    headless: 'new',
    executablePath: 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1440,900']
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900, deviceScaleFactor: 2 });

  console.log('[+] Navigating to http://localhost:4040...');
  await page.goto('http://localhost:4040', { waitUntil: 'domcontentloaded' });
  await sleep(1500);

  // 1. Desktop Dashboard
  console.log('[+] Capturing Dashboard...');
  await page.screenshot({ path: 'B:/workgit/ragfuzz/docs/images/ragfuzz_dashboard.png' });

  // 2. Query Inspector
  console.log('[+] Navigating to Query Inspector...');
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const btn = btns.find(b => b.textContent && b.textContent.includes('Query Inspector'));
    if (btn) btn.click();
  });
  await sleep(800);
  // Expand first failed test case row
  await page.evaluate(() => {
    const rows = Array.from(document.querySelectorAll('div, button'));
    const target = rows.find(el => el.textContent && el.textContent.includes('TC-DST-0001'));
    if (target) target.click();
  });
  await sleep(800);
  await page.screenshot({ path: 'B:/workgit/ragfuzz/docs/images/ragfuzz_inspector.png' });
  console.log('[+] Captured Query Inspector');

  // 3. Topology Graph
  console.log('[+] Navigating to Topology Graph...');
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const btn = btns.find(b => b.textContent && b.textContent.includes('Topology Graph'));
    if (btn) btn.click();
  });
  await sleep(1000);
  await page.screenshot({ path: 'B:/workgit/ragfuzz/docs/images/ragfuzz_topology.png' });
  console.log('[+] Captured Topology Graph');

  // 4. Live Playground
  console.log('[+] Navigating to Live Playground...');
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const btn = btns.find(b => b.textContent && b.textContent.includes('Live Playground'));
    if (btn) btn.click();
  });
  await sleep(800);
  // Click "Run Adversarial Probe"
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const btn = btns.find(b => b.textContent && b.textContent.includes('Run Adversarial Probe'));
    if (btn) btn.click();
  });
  await sleep(1500); // Wait for probe execution against mock target
  await page.screenshot({ path: 'B:/workgit/ragfuzz/docs/images/ragfuzz_playground.png' });
  console.log('[+] Captured Live Playground');

  // 5. Mobile Dashboard (390x844)
  console.log('[+] Navigating to Mobile Viewport...');
  await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 2, isMobile: true, hasTouch: true });
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const btn = btns.find(b => b.textContent && b.textContent.includes('Overview & Radar'));
    if (btn) btn.click();
  });
  await sleep(1000);
  await page.screenshot({ path: 'B:/workgit/ragfuzz/docs/images/ragfuzz_mobile.png' });
  console.log('[+] Captured Mobile Dashboard');

  await browser.close();
  console.log('[+] All static screenshots successfully captured!');
})();
