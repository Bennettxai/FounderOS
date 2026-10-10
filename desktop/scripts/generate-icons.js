const { createCanvas } = require('canvas');
const fs = require('fs');
const path = require('path');

// Icon sizes needed
const sizes = [16, 32, 64, 128, 256, 512, 1024];

const outputDir = path.join(__dirname, '../resources/icons');

// Ensure output directory exists
if (!fs.existsSync(outputDir)) {
  fs.mkdirSync(outputDir, { recursive: true });
}

function generateIcon(size) {
  const canvas = createCanvas(size, size);
  const ctx = canvas.getContext('2d');

  // Background - black
  ctx.fillStyle = '#000000';
  ctx.beginPath();
  ctx.roundRect(0, 0, size, size, size * 0.2);
  ctx.fill();

  // Inner slightly lighter area
  const padding = size * 0.08;
  ctx.fillStyle = '#111111';
  ctx.beginPath();
  ctx.roundRect(padding, padding, size - padding * 2, size - padding * 2, size * 0.15);
  ctx.fill();

  // Draw "B" letter
  ctx.fillStyle = '#ffffff';
  ctx.font = `bold ${size * 0.6}px -apple-system, BlinkMacSystemFont, "SF Pro Display", "Helvetica Neue", sans-serif`;
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  ctx.fillText('B', size / 2, size / 2 + size * 0.02);

  // Small "OS" subscript
  ctx.fillStyle = 'rgba(255, 255, 255, 0.4)';
  ctx.font = `500 ${size * 0.15}px -apple-system, BlinkMacSystemFont, "SF Pro Display", "Helvetica Neue", sans-serif`;
  ctx.fillText('OS', size * 0.72, size * 0.75);

  return canvas;
}

// Generate PNG icons at all sizes
console.log('Generating icons...');

sizes.forEach(size => {
  const canvas = generateIcon(size);
  const buffer = canvas.toBuffer('image/png');
  const filename = `icon-${size}.png`;
  fs.writeFileSync(path.join(outputDir, filename), buffer);
  console.log(`Created ${filename}`);
});

// Create main icon.png at 1024px
const mainIcon = generateIcon(1024);
fs.writeFileSync(path.join(outputDir, 'icon.png'), mainIcon.toBuffer('image/png'));
console.log('Created icon.png (1024x1024)');

// Create tray icon (smaller, template-ready)
const trayCanvas = createCanvas(22, 22);
const trayCtx = trayCanvas.getContext('2d');

// Transparent background for tray
trayCtx.clearRect(0, 0, 22, 22);

// Draw "B" in black (will be inverted by macOS for template)
trayCtx.fillStyle = '#000000';
trayCtx.font = 'bold 16px -apple-system, BlinkMacSystemFont, sans-serif';
trayCtx.textAlign = 'center';
trayCtx.textBaseline = 'middle';
trayCtx.fillText('B', 11, 12);

fs.writeFileSync(path.join(outputDir, 'tray-icon.png'), trayCanvas.toBuffer('image/png'));
console.log('Created tray-icon.png');

console.log('\nIcons generated successfully!');
console.log('\nTo create .icns (macOS) and .ico (Windows), use:');
console.log('  macOS: iconutil -c icns iconset.iconset');
console.log('  Windows: Use an online converter or png2ico tool');
