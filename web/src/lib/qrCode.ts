const VERSION = 5;
const SIZE = 21 + (VERSION - 1) * 4;
const DATA_CODEWORDS = 108;
const ECC_CODEWORDS = 26;
const MAX_BYTE_LENGTH = 106;
const FORMAT_POLY = 0x537;
const FORMAT_MASK = 0x5412;
const ECL_LOW_FORMAT_BITS = 1;

type Matrix = boolean[][];

interface QrMatrix {
  modules: Matrix;
  reserved: Matrix;
}

const gf = createGaloisField();

export function createQrCodeMatrix(value: string) {
  const data = encodeData(value);
  const base = createBaseMatrix();
  let bestModules: Matrix | undefined;
  let bestPenalty = Number.POSITIVE_INFINITY;

  for (let mask = 0; mask < 8; mask += 1) {
    const candidate = cloneMatrix(base.modules);
    placeCodewords(candidate, base.reserved, data);
    applyMask(candidate, base.reserved, mask);
    drawFormatBits(candidate, base.reserved, mask);

    const penalty = scorePenalty(candidate);
    if (penalty < bestPenalty) {
      bestPenalty = penalty;
      bestModules = candidate;
    }
  }

  return bestModules ?? base.modules;
}

export function getQrCodeSize() {
  return SIZE;
}

export function canEncodeQrCode(value: string) {
  return new TextEncoder().encode(value).length <= MAX_BYTE_LENGTH;
}

function encodeData(value: string) {
  const bytes = [...new TextEncoder().encode(value)];
  if (bytes.length > MAX_BYTE_LENGTH) {
    throw new Error(`QR payload is too long: ${bytes.length} bytes`);
  }

  const bits = [0, 1, 0, 0, ...toBits(bytes.length, 8), ...bytes.flatMap((byte) => toBits(byte, 8))];
  const capacityBits = DATA_CODEWORDS * 8;
  const terminatorLength = Math.min(4, capacityBits - bits.length);
  for (let i = 0; i < terminatorLength; i += 1) bits.push(0);
  while (bits.length % 8 !== 0) bits.push(0);

  const codewords: number[] = [];
  for (let i = 0; i < bits.length; i += 8) {
    codewords.push(parseInt(bits.slice(i, i + 8).join(""), 2));
  }

  for (let pad = 0xec; codewords.length < DATA_CODEWORDS; pad ^= 0xfd) {
    codewords.push(pad);
  }

  return [...codewords, ...computeErrorCorrection(codewords)];
}

function createBaseMatrix(): QrMatrix {
  const modules = createMatrix(false);
  const reserved = createMatrix(false);

  drawFinder(modules, reserved, 3, 3);
  drawFinder(modules, reserved, SIZE - 4, 3);
  drawFinder(modules, reserved, 3, SIZE - 4);
  drawAlignment(modules, reserved, 30, 30);
  drawTiming(modules, reserved);
  setModule(modules, reserved, 8, 4 * VERSION + 9, true);
  reserveFormatBits(reserved);

  return { modules, reserved };
}

function drawFinder(modules: Matrix, reserved: Matrix, centerX: number, centerY: number) {
  for (let y = centerY - 4; y <= centerY + 4; y += 1) {
    for (let x = centerX - 4; x <= centerX + 4; x += 1) {
      if (x < 0 || y < 0 || x >= SIZE || y >= SIZE) continue;
      const distance = Math.max(Math.abs(x - centerX), Math.abs(y - centerY));
      setModule(modules, reserved, x, y, distance === 3 || distance <= 1);
    }
  }
}

function drawAlignment(modules: Matrix, reserved: Matrix, centerX: number, centerY: number) {
  for (let y = centerY - 2; y <= centerY + 2; y += 1) {
    for (let x = centerX - 2; x <= centerX + 2; x += 1) {
      const distance = Math.max(Math.abs(x - centerX), Math.abs(y - centerY));
      setModule(modules, reserved, x, y, distance === 2 || distance === 0);
    }
  }
}

function drawTiming(modules: Matrix, reserved: Matrix) {
  for (let i = 8; i < SIZE - 8; i += 1) {
    setModule(modules, reserved, i, 6, i % 2 === 0);
    setModule(modules, reserved, 6, i, i % 2 === 0);
  }
}

function reserveFormatBits(reserved: Matrix) {
  for (let i = 0; i <= 5; i += 1) reserved[i][8] = true;
  reserved[7][8] = true;
  reserved[8][8] = true;
  reserved[8][7] = true;
  for (let i = 9; i < 15; i += 1) reserved[8][14 - i] = true;
  for (let i = 0; i < 8; i += 1) reserved[8][SIZE - 1 - i] = true;
  for (let i = 8; i < 15; i += 1) reserved[SIZE - 15 + i][8] = true;
}

function drawFormatBits(modules: Matrix, reserved: Matrix, mask: number) {
  const bits = getFormatBits(mask);
  for (let i = 0; i <= 5; i += 1) setModule(modules, reserved, 8, i, bitAt(bits, i));
  setModule(modules, reserved, 8, 7, bitAt(bits, 6));
  setModule(modules, reserved, 8, 8, bitAt(bits, 7));
  setModule(modules, reserved, 7, 8, bitAt(bits, 8));
  for (let i = 9; i < 15; i += 1) setModule(modules, reserved, 14 - i, 8, bitAt(bits, i));
  for (let i = 0; i < 8; i += 1) setModule(modules, reserved, SIZE - 1 - i, 8, bitAt(bits, i));
  for (let i = 8; i < 15; i += 1) setModule(modules, reserved, 8, SIZE - 15 + i, bitAt(bits, i));
  setModule(modules, reserved, 8, 4 * VERSION + 9, true);
}

function getFormatBits(mask: number) {
  const data = (ECL_LOW_FORMAT_BITS << 3) | mask;
  let remainder = data;
  for (let i = 0; i < 10; i += 1) {
    remainder = (remainder << 1) ^ (((remainder >>> 9) & 1) * FORMAT_POLY);
  }
  return ((data << 10) | remainder) ^ FORMAT_MASK;
}

function placeCodewords(modules: Matrix, reserved: Matrix, codewords: number[]) {
  const bits = codewords.flatMap((codeword) => toBits(codeword, 8));
  let bitIndex = 0;
  let upward = true;

  for (let right = SIZE - 1; right >= 1; right -= 2) {
    if (right === 6) right -= 1;
    for (let vertical = 0; vertical < SIZE; vertical += 1) {
      const y = upward ? SIZE - 1 - vertical : vertical;
      for (let offset = 0; offset < 2; offset += 1) {
        const x = right - offset;
        if (reserved[y][x]) continue;
        modules[y][x] = bitIndex < bits.length ? bits[bitIndex] === 1 : false;
        bitIndex += 1;
      }
    }
    upward = !upward;
  }
}

function applyMask(modules: Matrix, reserved: Matrix, mask: number) {
  for (let y = 0; y < SIZE; y += 1) {
    for (let x = 0; x < SIZE; x += 1) {
      if (!reserved[y][x] && maskApplies(mask, x, y)) {
        modules[y][x] = !modules[y][x];
      }
    }
  }
}

function maskApplies(mask: number, x: number, y: number) {
  switch (mask) {
    case 0:
      return (x + y) % 2 === 0;
    case 1:
      return y % 2 === 0;
    case 2:
      return x % 3 === 0;
    case 3:
      return (x + y) % 3 === 0;
    case 4:
      return (Math.floor(y / 2) + Math.floor(x / 3)) % 2 === 0;
    case 5:
      return ((x * y) % 2) + ((x * y) % 3) === 0;
    case 6:
      return (((x * y) % 2) + ((x * y) % 3)) % 2 === 0;
    case 7:
      return (((x + y) % 2) + ((x * y) % 3)) % 2 === 0;
    default:
      return false;
  }
}

function computeErrorCorrection(data: number[]) {
  const generator = createGenerator(ECC_CODEWORDS);
  const result = Array(ECC_CODEWORDS).fill(0);

  for (const codeword of data) {
    const factor = codeword ^ result.shift();
    result.push(0);
    for (let i = 0; i < generator.length; i += 1) {
      result[i] ^= multiply(generator[i], factor);
    }
  }

  return result;
}

function createGenerator(degree: number) {
  let result = [1];
  for (let i = 0; i < degree; i += 1) {
    const next = Array(result.length + 1).fill(0);
    for (let j = 0; j < result.length; j += 1) {
      next[j] ^= result[j];
      next[j + 1] ^= multiply(result[j], gf.exp[i]);
    }
    result = next;
  }
  return result.slice(1);
}

function createGaloisField() {
  const exp = Array(512).fill(0);
  const log = Array(256).fill(0);
  let value = 1;

  for (let i = 0; i < 255; i += 1) {
    exp[i] = value;
    log[value] = i;
    value <<= 1;
    if (value & 0x100) value ^= 0x11d;
  }
  for (let i = 255; i < 512; i += 1) exp[i] = exp[i - 255];

  return { exp, log };
}

function multiply(a: number, b: number) {
  if (a === 0 || b === 0) return 0;
  return gf.exp[gf.log[a] + gf.log[b]];
}

function scorePenalty(modules: Matrix) {
  return scoreRuns(modules) + scoreBlocks(modules) + scoreFinderPatterns(modules) + scoreDarkRatio(modules);
}

function scoreRuns(modules: Matrix) {
  let penalty = 0;
  const lines = [...modules, ...columns(modules)];

  for (const line of lines) {
    let runColor = line[0];
    let runLength = 1;
    for (let i = 1; i < line.length; i += 1) {
      if (line[i] === runColor) {
        runLength += 1;
      } else {
        if (runLength >= 5) penalty += 3 + runLength - 5;
        runColor = line[i];
        runLength = 1;
      }
    }
    if (runLength >= 5) penalty += 3 + runLength - 5;
  }

  return penalty;
}

function scoreBlocks(modules: Matrix) {
  let penalty = 0;
  for (let y = 0; y < SIZE - 1; y += 1) {
    for (let x = 0; x < SIZE - 1; x += 1) {
      const color = modules[y][x];
      if (modules[y][x + 1] === color && modules[y + 1][x] === color && modules[y + 1][x + 1] === color) {
        penalty += 3;
      }
    }
  }
  return penalty;
}

function scoreFinderPatterns(modules: Matrix) {
  let penalty = 0;
  const lines = [...modules, ...columns(modules)];
  const pattern = [true, false, true, true, true, false, true, false, false, false, false];
  const reverse = [...pattern].reverse();

  for (const line of lines) {
    for (let i = 0; i <= line.length - pattern.length; i += 1) {
      const window = line.slice(i, i + pattern.length);
      if (matchesPattern(window, pattern) || matchesPattern(window, reverse)) penalty += 40;
    }
  }

  return penalty;
}

function scoreDarkRatio(modules: Matrix) {
  const dark = modules.flat().filter(Boolean).length;
  const total = SIZE * SIZE;
  return Math.floor(Math.abs(dark * 20 - total * 10) / total) * 10;
}

function columns(modules: Matrix) {
  return Array.from({ length: SIZE }, (_, x) => modules.map((row) => row[x]));
}

function matchesPattern(values: boolean[], pattern: boolean[]) {
  return values.every((value, index) => value === pattern[index]);
}

function createMatrix(value: boolean) {
  return Array.from({ length: SIZE }, () => Array(SIZE).fill(value));
}

function cloneMatrix(matrix: Matrix) {
  return matrix.map((row) => [...row]);
}

function setModule(modules: Matrix, reserved: Matrix, x: number, y: number, value: boolean) {
  modules[y][x] = value;
  reserved[y][x] = true;
}

function toBits(value: number, length: number) {
  return Array.from({ length }, (_, index) => (value >>> (length - 1 - index)) & 1);
}

function bitAt(value: number, index: number) {
  return ((value >>> index) & 1) !== 0;
}
