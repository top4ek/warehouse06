// Intel 8080 (KR580VM80A) microprocessor core model in JavaScript
//
// Copyright (C) 2012 Alexander Demin <alexander@demin.ws>
//
// Credits
//
// Viacheslav Slavinsky, Vector-06C FPGA Replica
// http://code.google.com/p/vector06cc/
//
// Dmitry Tselikov, Bashrikia-2M and Radio-86RK on Altera DE1
// http://bashkiria-2m.narod.ru/fpga.html
//
// Ian Bartholomew, 8080/8085 CPU Exerciser
// http://www.idb.me.uk/sunhillow/8080.html
//
// Frank Cringle, The original exerciser for the Z80.
//
// Thanks to zx.pk.ru and nedopc.org/forum communities.
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2, or (at your option)
// any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 59 Temple Place - Suite 330, Boston, MA 02111-1307, USA.

/** @constructor */
function I8080(memory, io) {
  this.sp = 0;
  this.pc = 0;
  this.iff = false;
  this.iff_pending = 0;

  this.sf = 0;
  this.pf = 0;
  this.hf = 0;
  this.zf = 0;
  this.cf = 0;

  // Registers: b, c, d, e, h, l, m, a
  //            0  1  2  3  4  5  6  7
  this.regs = [ 0, 0, 0, 0, 0, 0, 0, 0 ];

  this.memory = memory;
  this.io = io;

  this.tstates = [];

  this.last_opcode = 0;

  this.initHandlers();

  const F_CARRY  = 0x01;
  const F_UN1    = 0x02;
  const F_PARITY = 0x04;
  const F_UN3    = 0x08;
  const F_HCARRY = 0x10;
  const F_UN5    = 0x20;
  const F_ZERO   = 0x40;
  const F_NEG    = 0x80;

  I8080.prototype.store_flags = function() {
    var f = 0;
    if (this.sf) f |= F_NEG;    else f &= ~F_NEG;
    if (this.zf) f |= F_ZERO;   else f &= ~F_ZERO;
    if (this.hf) f |= F_HCARRY; else f &= ~F_HCARRY;
    if (this.pf) f |= F_PARITY; else f &= ~F_PARITY;
    if (this.cf) f |= F_CARRY;  else f &= ~F_CARRY;
    f |= F_UN1;    // UN1_FLAG is always 1.
    f &= ~F_UN3;   // UN3_FLAG is always 0.
    f &= ~F_UN5;   // UN5_FLAG is always 0.
    return f;
  }

  I8080.prototype.retrieve_flags = function(f) {
    this.sf = f & F_NEG    ? 1 : 0;
    this.zf = f & F_ZERO   ? 1 : 0;
    this.hf = f & F_HCARRY ? 1 : 0;
    this.pf = f & F_PARITY ? 1 : 0;
    this.cf = f & F_CARRY  ? 1 : 0;
  }
}

I8080.prototype.memory_read_byte = function(addr, stackrq) {
  return this.memory.read(addr & 0xffff, stackrq) & 0xff;
}

I8080.prototype.memory_write_byte = function(addr, w8, stackrq) {
  this.memory.write(addr & 0xffff, w8 & 0xff, stackrq);
}

I8080.prototype.memory_read_word = function(addr, stackrq) {
  return this.memory_read_byte(addr, stackrq) | (this.memory_read_byte(addr + 1, stackrq) << 8); 
}

I8080.prototype.memory_write_word = function(addr, w16, stackrq) {
  this.memory_write_byte(addr, w16 & 0xff, stackrq);
  this.memory_write_byte(addr + 1, w16 >> 8, stackrq);
}

I8080.prototype.reg = function(r) {
  return r != 6 ? this.regs[r] : this.memory_read_byte(this.hl());
}

I8080.prototype.set_reg = function(r, w8) {
  w8 &= 0xff;
  if (r != 6)
    this.regs[r] = w8;
  else
    this.memory_write_byte(this.hl(), w8);
}

// r - 00 (bc), 01 (de), 10 (hl), 11 (sp)
I8080.prototype.rp = function(r) {
  return r != 6 ? (this.regs[r] << 8) | this.regs[r + 1] : this.sp;
}

I8080.prototype.set_rp = function(r, w16) {
  if (r != 6) {
    this.set_reg(r, w16 >> 8);
    this.set_reg(r + 1, w16 & 0xff);
  } else
    this.sp = w16;
}
I8080.prototype.parity_table = [
  1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
  0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
  0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
  1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
  0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
  1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
  1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
  0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
  0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
  1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
  1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
  0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
  1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
  0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
  0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0,
  1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1,
];
I8080.prototype.half_carry_table = [ 0, 0, 1, 0, 1, 0, 1, 1 ];
I8080.prototype.sub_half_carry_table = [ 0, 1, 1, 1, 0, 0, 0, 1 ];

I8080.prototype.bc = function() { return this.rp(0); }
I8080.prototype.de = function() { return this.rp(2); }
I8080.prototype.hl = function() { return this.rp(4); }

I8080.prototype.b = function() { return this.reg(0); }
I8080.prototype.c = function() { return this.reg(1); }
I8080.prototype.d = function() { return this.reg(2); }
I8080.prototype.e = function() { return this.reg(3); }
I8080.prototype.h = function() { return this.reg(4); }
I8080.prototype.l = function() { return this.reg(5); }
I8080.prototype.a = function() { return this.reg(7); }

I8080.prototype.set_b = function(v) { this.set_reg(0, v); }
I8080.prototype.set_c = function(v) { this.set_reg(1, v); }
I8080.prototype.set_d = function(v) { this.set_reg(2, v); }
I8080.prototype.set_e = function(v) { this.set_reg(3, v); }
I8080.prototype.set_h = function(v) { this.set_reg(4, v); }
I8080.prototype.set_l = function(v) { this.set_reg(5, v); }
I8080.prototype.set_a = function(v) { this.set_reg(7, v); }

I8080.prototype.next_pc_byte = function() {
  var v = this.memory_read_byte(this.pc);
  this.pc = (this.pc + 1) & 0xffff;
  return v;
}

I8080.prototype.next_pc_word = function() {
  return this.next_pc_byte() | (this.next_pc_byte() << 8);
}

I8080.prototype.inr = function(r) {
  var v = this.reg(r);
  v = (v + 1) & 0xff;
  this.set_reg(r, v);
  this.sf = (v & 0x80) != 0;
  this.zf = (v == 0);
  this.hf = (v & 0x0f) == 0;
  this.pf = I8080.prototype.parity_table[v];
}

I8080.prototype.dcr = function(r) {
  var v = this.reg(r);
  v = (v - 1) & 0xff;
  this.set_reg(r, v);
  this.sf = (v & 0x80) != 0;
  this.zf = (v == 0);
  this.hf = !((v & 0x0f) == 0x0f);
  this.pf = I8080.prototype.parity_table[v];
}

I8080.prototype.add_im8 = function(v, carry) {
  var a = this.a();
  var w16 = a + v + carry;
  var index = ((a & 0x88) >> 1) | ((v & 0x88) >> 2) | ((w16 & 0x88) >> 3);
  a = w16 & 0xff;
  this.sf = (a & 0x80) != 0;
  this.zf = (a == 0);
  this.hf = I8080.prototype.half_carry_table[index & 0x7];
  this.pf = I8080.prototype.parity_table[a];
  this.cf = (w16 & 0x0100) != 0;
  this.set_a(a);
}

I8080.prototype.add = function(r, carry) {
  this.add_im8(this.reg(r), carry);
}

I8080.prototype.sub_im8 = function(v, carry) {
  var a = this.a();
  var w16 = (a - v - carry) & 0xffff;
  var index = ((a & 0x88) >> 1) | ((v & 0x88) >> 2) | ((w16 & 0x88) >> 3);
  a = w16 & 0xff;
  this.sf = (a & 0x80) != 0;
  this.zf = (a == 0);
  this.hf = !I8080.prototype.sub_half_carry_table[index & 0x7];
  this.pf = I8080.prototype.parity_table[a];
  this.cf = (w16 & 0x0100) != 0;
  this.set_a(a);
}

I8080.prototype.sub = function(r, carry) {
  this.sub_im8(this.reg(r), carry);
}

I8080.prototype.cmp_im8 = function(v) {
  var a = this.a();    // Store the accumulator before substraction.
  this.sub_im8(v, 0);
  this.set_a(a);       // Ignore the accumulator value after substraction.
}

I8080.prototype.cmp = function(r) {
  this.cmp_im8(this.reg(r));
}

I8080.prototype.ana_im8 = function(v) {
  var a = this.a();
  this.hf = ((a | v) & 0x08) != 0;
  a &= v;
  this.sf = (a & 0x80) != 0;
  this.zf = (a == 0);
  this.pf = I8080.prototype.parity_table[a];
  this.cf = 0;
  this.set_a(a);
}

I8080.prototype.ana = function(r) {
  this.ana_im8(this.reg(r));
}

I8080.prototype.xra_im8 = function(v) {
  var a = this.a();
  a ^= v;
  this.sf = (a & 0x80) != 0;
  this.zf = (a == 0);
  this.hf = 0;
  this.pf = I8080.prototype.parity_table[a];
  this.cf = 0;
  this.set_a(a);
}

I8080.prototype.xra = function(r) {
  this.xra_im8(this.reg(r));
}

I8080.prototype.ora_im8 = function(v) {
  var a = this.a();
  a |= v;
  this.sf = (a & 0x80) != 0;
  this.zf = (a == 0);
  this.hf = 0;
  this.pf = I8080.prototype.parity_table[a];
  this.cf = 0;
  this.set_a(a);
}

I8080.prototype.ora = function(r) {
  this.ora_im8(this.reg(r));
}

// r - 0 (bc), 2 (de), 4 (hl), 6 (sp)
I8080.prototype.dad = function(r) {
  var hl = this.hl() + this.rp(r);
  this.cf = (hl & 0x10000) != 0;
  this.set_h(hl >> 8);
  this.set_l(hl & 0xff);
}

I8080.prototype.call = function(w16) {
  this.push(this.pc);
  this.pc = w16;
}

I8080.prototype.ret = function() {
  this.pc = this.pop();
}

I8080.prototype.pop = function() {
  var v = this.memory_read_word(this.sp, true);
  this.sp = (this.sp + 2) & 0xffff;
  return v;
}

I8080.prototype.push = function(v) {
  this.sp = (this.sp - 2) & 0xffff;
  this.memory_write_word(this.sp, v, true);
}

I8080.prototype.rst = function(addr) {
  this.push(this.pc);
  this.pc = addr;
}

const T4 = 4;  // 4
const T43 = 8; // 4, 3
const T5  = 8  // 5
const T433 = 12; // 4, 3, 3
const T43333 = 20; 
const T4333 = 16;
const T533 = 16;
const T53333 = 24;
const T43335 = 24;

I8080.prototype.initHandlers = function() {
    var handler = [];
    // nop, 0x00, 00rrr000
    // r - 000(0) to 111(7)
    handler[0x00] = 
    // Undocumented NOP.
    handler[0x08] =            /* nop */
    handler[0x10] =            /* nop */
    handler[0x18] =            /* nop */
    handler[0x20] =            /* nop */
    handler[0x28] =            /* nop */
    handler[0x30] =            /* nop */
    handler[0x38] =            /* nop */
        (function(that) {
            return function() {
              that.vcycles = T4;
              that.cpu_cycles = 4;
            };
        })(this);
   
    // lxi, 0x01, 00rr0001
    // rr - 00 (bc), 01 (de), 10 (hl), 11 (sp)
    handler[0x01] =            /* lxi b, data16 */
    handler[0x11] =            /* lxi d, data16 */
    handler[0x21] =            /* lxi h, data16 */
    handler[0x31] =            /* lxi sp, data16 */
        (function(that) {
          return function(opcode) {
              that.vcycles = T433;
              that.cpu_cycles = 10;
              that.set_rp(opcode >> 3, that.next_pc_word());
          };
        })(this);

    // stax, 0x02, 000r0010
    // r - 0 (bc), 1 (de)
    handler[0x02] =            /* stax b */
    handler[0x12] =            /* stax d */
        (function(that) {
                return function(opcode) {
                  that.vcycles = T43;
                  that.cpu_cycles = 7;
                  that.memory_write_byte(that.rp(opcode >> 3), that.a());
                };
        })(this);


    // inx, 0x03, 00rr0011
    // rr - 00 (bc), 01 (de), 10 (hl), 11 (sp)
    handler[0x03] =            /* inx b */
    handler[0x13] =            /* inx d */
    handler[0x23] =            /* inx h */
    handler[0x33] =            /* inx sp */
        (function(that) {
          return function(opcode) {
              that.vcycles = T5;
              that.cpu_cycles = 5;
              var r = opcode >> 3;
              that.set_rp(r, (that.rp(r) + 1) & 0xffff);
          };
        })(this);

    // inr, 0x04, 00rrr100
    // rrr - b, c, d, e, h, l, m, a
    handler[0x04] =            /* inr b */
    handler[0x0C] =            /* inr c */
    handler[0x14] =            /* inr d */
    handler[0x1C] =            /* inr e */
    handler[0x24] =            /* inr h */
    handler[0x2C] =            /* inr l */
    handler[0x3C] =            /* inr a */
        (function(that) {
          return function(opcode) {
              that.vcycles = T5;
              that.cpu_cycles = 5;
              that.inr(opcode >> 3);
          };
        })(this);
    handler[0x34] =            /* inr m */
        (function(that) {
          return function(opcode) {
              that.vcycles = T433;
              that.cpu_cycles = 10;
              that.inr(opcode >> 3);
          };
        })(this);

    // dcr, 0x05, 00rrr100
    // rrr - b, c, d, e, h, l, m, a
    handler[0x05] =            /* dcr b */
    handler[0x0D] =            /* dcr c */
    handler[0x15] =            /* dcr d */
    handler[0x1D] =            /* dcr e */
    handler[0x25] =            /* dcr h */
    handler[0x2D] =            /* dcr l */
    handler[0x3D] =            /* dcr a */
        (function(that) {
          return function(opcode) {
              that.vcycles = T5;
              that.cpu_cycles = 5;
              that.dcr(opcode >> 3);
          };
        })(this);
    handler[0x35] =            /* dcr m */
        (function(that) {
          return function(opcode) {
              that.vcycles = T433;
              that.cpu_cycles = 10;
              that.dcr(opcode >> 3);
          };
        })(this);

    // mvi, 0x06, 00rrr110
    // rrr - b, c, d, e, h, l, m, a
    handler[0x06] =       /* mvi b, data8 */
    handler[0x0E] =       /* mvi c, data8 */
    handler[0x16] =       /* mvi d, data8 */
    handler[0x1E] =       /* mvi e, data8 */
    handler[0x26] =       /* mvi h, data8 */
    handler[0x2E] =       /* mvi l, data8 */
    handler[0x3E] =       /* mvi a, data8 */
        (function(that) {
          return function(opcode) {
              that.vcycles = T43;
              that.cpu_cycles = 7;
              that.set_reg(opcode >> 3, that.next_pc_byte());
          };
        })(this);

    handler[0x36] =       /* mvi m, data8 */
        (function(that) {
          return function(opcode) {
              that.vcycles = T433;
              that.cpu_cycles = 10;
              that.set_reg(opcode >> 3, that.next_pc_byte());
          };
        })(this);

    handler[0x07] =       /* rlc */
        (function(that) {
          return function() {
              that.vcycles = T4;
              that.cpu_cycles = 4;
              var a = that.a();
              that.cf = ((a & 0x80) != 0);
              that.set_a(((a << 1) & 0xff) | that.cf);
          };
        })(this);

    // dad, 0x09, 00rr1001
    // rr - 00 (bc), 01 (de), 10 (hl), 11 (sp)
    handler[0x09] =       /* dad b */
    handler[0x19] =       /* dad d */
    handler[0x29] =       /* dad hl */
    handler[0x39] =       /* dad sp */
        (function(that) {
          return function(opcode) {
              that.vcycles = T433;
              that.cpu_cycles = 10;
              that.dad((opcode & 0x30) >> 3);
          };
        })(this);

    // ldax, 0x0A, 000r1010
    // r - 0 (bc), 1 (de)
    handler[0x0A] =       /* ldax b */
    handler[0x1A] =       /* ldax d */
        (function(that) {
          return function(opcode) {
              that.vcycles = T43;
              that.cpu_cycles = 7;
              var r = (opcode & 0x10) >> 3;
              that.set_a(that.memory_read_byte(that.rp(r)));
          };
        })(this);

    // dcx, 0x0B, 00rr1011
    // rr - 00 (bc), 01 (de), 10 (hl), 11 (sp)
    handler[0x0B] =       /* dcx b */
    handler[0x1B] =       /* dcx d */
    handler[0x2B] =       /* dcx h */
    handler[0x3B] =       /* dcx sp */
        (function(that) {
          return function(opcode) {
              that.vcycles = T5;
              that.cpu_cycles = 5;
              var r = (opcode & 0x30) >> 3;
              that.set_rp(r, (that.rp(r) - 1) & 0xffff);
          };
        })(this);

    handler[0x0F] =       /* rrc */
        (function(that) {
          return function() {
              that.vcycles = T4;
              that.cpu_cycles = 4;
              that.cf = that.a() & 0x01;
              that.set_a((that.a() >> 1) | (that.cf << 7));
          };
        })(this);

    handler[0x17] =       /* ral */
        (function(that) {
          return function() {
              that.vcycles = T4;
              that.cpu_cycles = 4;
              var w8 = that.cf;
              that.cf = ((that.a() & 0x80) != 0);
              that.set_a((that.a() << 1) | w8);
          };
        })(this);

    handler[0x1F] =        /* rar */
        (function(that) {
          return function() {
              that.vcycles = T4;
              that.cpu_cycles = 4;
              var w8 = that.cf;
              that.cf = that.a() & 0x01;
              that.set_a((that.a() >> 1) | (w8 << 7));
          };
        })(this);

    handler[0x22] =       /* shld addr */
        (function(that) {
          return function() {
              that.vcycles = T43333;
              that.cpu_cycles = 16;
              var w16 = that.next_pc_word();
              that.memory_write_byte(w16, that.l());
              that.memory_write_byte(w16 + 1, that.h());
          };
        })(this);

    handler[0x27] =       /* daa */
        (function(that) {
          return function() {
              that.vcycles = T4;
              that.cpu_cycles = 4;
              var carry = that.cf;
              var add = 0;
              var a = that.a();
              if (that.hf || (a & 0x0f) > 9) add = 0x06;
              if (that.cf || (a >> 4) > 9 || ((a >> 4) >= 9 && (a & 0xf) > 9)) {
                  add |= 0x60;
                  carry = 1;
              }
              that.add_im8(add, 0);
              that.pf = I8080.prototype.parity_table[that.a()];
              that.cf = carry;
          };
        })(this);

    handler[0x2A] =       /* lhld addr */
        (function(that) {
          return function() {
              that.vcycles = T43333;
              that.cpu_cycles = 16;
              var w16 = that.next_pc_word();
              that.regs[5] = that.memory_read_byte(w16);
              that.regs[4] = that.memory_read_byte(w16 + 1);
          };
        })(this);

    handler[0x2F] =       /* cma */
        (function(that) {
          return function() {
              that.vcycles = T4;
              that.cpu_cycles = 4;
              that.set_a(that.a() ^ 0xff);
          };
        })(this);

    handler[0x32] =       /* sta addr */
        (function(that) {
          return function() {
              that.vcycles = T4333;
              that.cpu_cycles = 13;
              that.memory_write_byte(that.next_pc_word(), that.a());
          };
        })(this);

    handler[0x37] =       /* stc */
        (function(that) {
          return function() {
              that.vcycles = T4;
              that.cpu_cycles = 4;
              that.cf = 1;
          };
        })(this);

    handler[0x3A] =       /* lda addr */
        (function(that) {
          return function() {
              that.vcycles = T4333;
              that.cpu_cycles = 13;
              that.set_a(that.memory_read_byte(that.next_pc_word()));
          };
        })(this);

    handler[0x3F] =       /* cmc */
        (function(that) {
          return function() {
              that.vcycles = T4;
              that.cpu_cycles = 4;
              that.cf = !that.cf;
          };
        })(this);
    // mov group moved out of the switch

    handler[0x76] =       /* hlt */
      // it should be that:
      // tstates = [4, 3];
      // that.vcycles = T43;
      // that.cpu_cycles = 7;
        (function(that) {
          return function() {
              that.vcycles = T4;
              that.cpu_cycles = 4;
              that.pc = (that.pc - 1) & 0xffff;
          };
        })(this);

    // add, 0x80, 10000rrr
    // rrr - b, c, d, e, h, l, m, a
    handler[0x80] =       /* add b */
    handler[0x81] =       /* add c */
    handler[0x82] =       /* add d */
    handler[0x83] =       /* add e */
    handler[0x84] =       /* add h */
    handler[0x85] =       /* add l */
    handler[0x87] =       /* add a */

    // adc, 0x80, 10001rrr
    // rrr - b, c, d, e, h, l, m, a
    handler[0x88] =       /* adc b */
    handler[0x89] =       /* adc c */
    handler[0x8A] =       /* adc d */
    handler[0x8B] =       /* adc e */
    handler[0x8C] =       /* adc h */
    handler[0x8D] =       /* adc l */
    handler[0x8F] =       /* adc a */
        (function(that) {
          return function(opcode) {
              var r = opcode & 0x07;          
              that.vcycles = T4;
              that.cpu_cycles = 4;
              that.add(r, (opcode & 0x08 ? that.cf : 0));
          };
        })(this);
    handler[0x86] =       /* add m */
    handler[0x8E] =       /* adc m */
        (function(that) {
          return function(opcode) {
              var r = opcode & 0x07; 
              that.vcycles = T43;         
              that.cpu_cycles = 7;
              that.add(r, (opcode & 0x08 ? that.cf : 0));
          };
        })(this);

    // sub, 0x90, 10010rrr
    // rrr - b, c, d, e, h, l, m, a
    handler[0x90] =       /* sub b */
    handler[0x91] =       /* sub c */
    handler[0x92] =       /* sub d */
    handler[0x93] =       /* sub e */
    handler[0x94] =       /* sub h */
    handler[0x95] =       /* sub l */
    handler[0x97] =       /* sub a */

    // sbb, 0x98, 10010rrr
    // rrr - b, c, d, e, h, l, m, a
    handler[0x98] =       /* sbb b */
    handler[0x99] =       /* sbb c */
    handler[0x9A] =       /* sbb d */
    handler[0x9B] =       /* sbb e */
    handler[0x9C] =       /* sbb h */
    handler[0x9D] =       /* sbb l */
    handler[0x9F] =       /* sbb a */
        (function(that) {
            return function(opcode) {
              var r = opcode & 0x07;
              that.vcycles = T4;
              that.cpu_cycles = 4;
              that.sub(r, (opcode & 0x08 ? that.cf : 0));
            };
        })(this);

    handler[0x96] =       /* sub m */
    handler[0x9E] =       /* sbb m */
        (function(that) {
          return function(opcode) {
              var r = opcode & 0x07;
              that.vcycles = T43;         
              that.cpu_cycles = 7;
              that.sub(r, (opcode & 0x08 ? that.cf : 0));
          };
        })(this);

    handler[0xA0] =       /* ana b */
    handler[0xA1] =       /* ana c */
    handler[0xA2] =       /* ana d */
    handler[0xA3] =       /* ana e */
    handler[0xA4] =       /* ana h */
    handler[0xA5] =       /* ana l */
    handler[0xA7] =       /* ana a */
        (function(that) {
          return function(opcode) {
              var r = opcode & 0x07;
              that.vcycles = T4;
              that.cpu_cycles = 4;
              that.ana(r);
          };
        })(this);

    handler[0xA6] =       /* ana m */
        (function(that) {
          return function(opcode) {
              var r = opcode & 0x07;
              that.vcycles = T43;         
              that.cpu_cycles = 7;
              that.ana(r);
          };
        })(this);

    handler[0xA8] =       /* xra b */
    handler[0xA9] =       /* xra c */
    handler[0xAA] =       /* xra d */
    handler[0xAB] =       /* xra e */
    handler[0xAC] =       /* xra h */
    handler[0xAD] =       /* xra l */
    handler[0xAF] =       /* xra a */
        (function(that) {
          return function(opcode) {
              var r = opcode & 0x07;
              that.vcycles = T4;
              that.cpu_cycles = 4;
              that.xra(r);
          };
        })(this);

    handler[0xAE] =       /* xra m */
        (function(that) {
          return function(opcode) {
              var r = opcode & 0x07;
              that.vcycles = T43;         
              that.cpu_cycles = 7;
              that.xra(r);
          };
        })(this);

    handler[0xB0] =       /* ora b */
    handler[0xB1] =       /* ora c */
    handler[0xB2] =       /* ora d */
    handler[0xB3] =       /* ora e */
    handler[0xB4] =       /* ora h */
    handler[0xB5] =       /* ora l */
    handler[0xB7] =       /* ora a */
        (function(that) {
          return function(opcode) {
              var r = opcode & 0x07;
              that.vcycles = T4;
              that.cpu_cycles = 4;
              that.ora(r);
          };
        })(this);
    handler[0xB6] =       /* ora m */
        (function(that) {
          return function(opcode) {
              var r = opcode & 0x07;
              that.vcycles = T43;         
              that.cpu_cycles = 7;
              that.ora(r);
          };
        })(this);

    handler[0xB8] =       /* cmp b */
    handler[0xB9] =       /* cmp c */
    handler[0xBA] =       /* cmp d */
    handler[0xBB] =       /* cmp e */
    handler[0xBC] =       /* cmp h */
    handler[0xBD] =       /* cmp l */
    handler[0xBF] =       /* cmp a */
        (function(that) {
          return function(opcode) {
              var r = opcode & 0x07;
              that.vcycles = T4;
              that.cpu_cycles = 4;
              that.cmp(r);
          };
        })(this);
    handler[0xBE] =       /* cmp m */
        (function(that) {
          return function(opcode) {
              var r = opcode & 0x07;
              that.vcycles = T43;         
              that.cpu_cycles = 7;
              that.cmp(r);
          };
        })(this);

    // rnz, rz, rnc, rc, rpo, rpe, rp, rm
    // 0xC0, 11ccd000
    // cc - 00 (zf), 01 (cf), 10 (pf), 11 (sf)
    // d - 0 (negate) or 1.
    handler[0xC0] =       /* rnz */
    handler[0xC8] =       /* rz */
    handler[0xD0] =       /* rnc */
    handler[0xD8] =       /* rc */
    handler[0xE0] =       /* rpo */
    handler[0xE8] =       /* rpe */
    handler[0xF0] =       /* rp */
    handler[0xF8] =       /* rm */
        (function(that) {
          return function(opcode) {
              const flags = [that.zf, that.cf, that.pf, that.sf];
              const r = (opcode >> 4) & 0x03;
              const direction = (opcode & 0x08) != 0;
              that.vcycles = T5;
              that.cpu_cycles = 5;
              if (flags[r] == direction) {
                that.cpu_cycles = 11;
                that.vcycles = T533;
                that.ret();
              }
          };
        })(this);

    // pop, 0xC1, 11rr0001
    // rr - 00 (bc), 01 (de), 10 (hl), 11 (psw)
    handler[0xC1] =       /* pop b */
    handler[0xD1] =       /* pop d */
    handler[0xE1] =       /* pop h */
    handler[0xF1] =       /* pop psw */
        (function(that) {
          return function(opcode) {
              const r = (opcode & 0x30) >> 3;
              that.vcycles = T433;
              that.cpu_cycles = 10;
              const w16 = that.pop();
              if (r != 6) {
                that.set_rp(r, w16);
              } else {
                that.set_a(w16 >> 8);
                that.retrieve_flags(w16 & 0xff);
              }
          };
        })(this);

    // jnz, jz, jnc, jc, jpo, jpe, jp, jm
    // 0xC2, 11ccd010
    // cc - 00 (zf), 01 (cf), 10 (pf), 11 (sf)
    // d - 0 (negate) or 1.
    handler[0xC2] =       /* jnz addr */
    handler[0xCA] =       /* jz addr */
    handler[0xD2] =       /* jnc addr */
    handler[0xDA] =       /* jc addr */
    handler[0xE2] =       /* jpo addr */
    handler[0xEA] =       /* jpe addr */
    handler[0xF2] =       /* jp addr */
    handler[0xFA] =       /* jm addr */
        (function(that) {
          return function(opcode) {
              const flags = [that.zf, that.cf, that.pf, that.sf];
              const r = (opcode >> 4) & 0x03;
              const direction = (opcode & 0x08) != 0;
              that.vcycles = T433;
              that.cpu_cycles = 10;
              const w16 = that.next_pc_word();
              that.pc = flags[r] == direction ? w16 : that.pc;
          };
        })(this);

    // jmp, 0xc3, 1100r011
    handler[0xC3] =       /* jmp addr */
    handler[0xCB] =       /* jmp addr, undocumented */
        (function(that) {
          return function() {
              that.vcycles = T433;
              that.cpu_cycles = 10;
              that.pc = that.next_pc_word();
          };
        })(this);

    // cnz, cz, cnc, cc, cpo, cpe, cp, cm
    // 0xC4, 11ccd100
    // cc - 00 (zf), 01 (cf), 10 (pf), 11 (sf)
    // d - 0 (negate) or 1.
    handler[0xC4] =       /* cnz addr */
    handler[0xCC] =       /* cz addr */
    handler[0xD4] =       /* cnc addr */
    handler[0xDC] =       /* cc addr */
    handler[0xE4] =       /* cpo addr */
    handler[0xEC] =       /* cpe addr */
    handler[0xF4] =       /* cp addr */
    handler[0xFC] =       /* cm addr */
        (function(that) {
          return function(opcode) {
              const flags = [that.zf, that.cf, that.pf, that.sf];
              const r = (opcode >> 4) & 0x03;
              const direction = (opcode & 0x08) != 0;
              const w16 = that.next_pc_word();
              that.vcycles = T533;
              that.cpu_cycles = 11;
              if (flags[r] == direction) {
                that.vcycles = T53333;
                that.cpu_cycles = 17;
                that.call(w16);
              }
          };
        })(this);

    // push, 0xC5, 11rr0101
    // rr - 00 (bc), 01 (de), 10 (hl), 11 (psw)
    handler[0xC5] =       /* push b */
    handler[0xD5] =       /* push d */
    handler[0xE5] =       /* push h */
    handler[0xF5] =       /* push psw */
        (function(that) {
          return function(opcode) {
              const r = (opcode & 0x30) >> 3;
              that.vcycles = T533;
              that.cpu_cycles = 11;
              const w16 = r != 6 ? that.rp(r) : (that.a() << 8) | that.store_flags();
              that.push(w16);
          };
        })(this);

    handler[0xC6] =       /* adi data8 */
        (function(that) {
          return function() {
              that.vcycles = T43;
              that.cpu_cycles = 7;
              that.add_im8(that.next_pc_byte(), 0);
          };
        })(this);

    // rst, 0xC7, 11aaa111
    // aaa - 000(0)-111(7), address = aaa*8 (0 to 0x38).
    handler[0xC7] =       /* rst 0 */
    handler[0xCF] =       /* rst 1 */
    handler[0xD7] =       /* rst 2 */
    handler[0xDF] =       /* rst 3 */
    handler[0xE7] =       /* rst 4 */
    handler[0xEF] =       /* rst 5 */
    handler[0xF7] =       /* rst 5 */
    handler[0xFF] =       /* rst 7 */
        (function(that) {
          return function(opcode) {
              that.vcycles = T533;
              that.cpu_cycles = 11;
              that.rst(opcode & 0x38);
          };
        })(this);

    // ret, 0xc9, 110r1001
    handler[0xC9] =       /* ret */
    handler[0xD9] =       /* ret, undocumented */
        (function(that) {
          return function() {
              that.vcycles = T433;
              that.cpu_cycles = 10;
              that.ret();
          };
        })(this);

    // call, 0xcd, 11rr1101
    handler[0xCD] =       /* call addr */
    handler[0xDD] =       /* call, undocumented */
    handler[0xED] = 
    handler[0xFD] = 
        (function(that) {
          return function() {
              that.vcycles = T53333;
              that.cpu_cycles = 17;
              that.call(that.next_pc_word());
          };
        })(this);

    handler[0xCE] =       /* aci data8 */
        (function(that) {
          return function() {
              that.vcycles = T43;
              that.cpu_cycles = 7;
              that.add_im8(that.next_pc_byte(), that.cf);
          };
        })(this);

    handler[0xD3] =       /* out port8 */
        (function(that) {
          return function() {
              that.vcycles = T433;
              that.cpu_cycles = 10;
              that.io.output(that.next_pc_byte(), that.a());
          };
        })(this);

    handler[0xD6] =       /* sui data8 */
        (function(that) {
          return function() {
              that.vcycles = T43;
              that.cpu_cycles = 7;
              that.sub_im8(that.next_pc_byte(), 0);
          };
        })(this);

    handler[0xDB] =       /* in port8 */
        (function(that) {
          return function() {
              that.vcycles = T433;
              that.cpu_cycles = 10;
              that.set_a(that.io.input(that.next_pc_byte()));
          };
        })(this);

    handler[0xDE] =       /* sbi data8 */
        (function(that) {
          return function() {
              that.vcycles = T43;
              that.cpu_cycles = 7;
              that.sub_im8(that.next_pc_byte(), that.cf);
          };
        })(this);

    handler[0xE3] =       /* xthl */
        (function(that) {
          return function() {
              that.vcycles = T43335;
              that.cpu_cycles = 18;
              const w16 = that.memory_read_word(that.sp, true);
              that.memory_write_word(that.sp, that.hl(), true);
              that.set_l(w16 & 0xff);
              that.set_h(w16 >> 8);
          };
        })(this);

    handler[0xE6] =       /* ani data8 */
        (function(that) {
          return function() {
              that.vcycles = T43;
              that.cpu_cycles = 7;
              that.ana_im8(that.next_pc_byte());
          };
        })(this);

    handler[0xE9] =       /* pchl */
        (function(that) {
          return function() {
              that.vcycles = T5;
              that.cpu_cycles = 5;
              that.pc = that.hl();
          };
        })(this);

    handler[0xEB] =       /* xchg */
        (function(that) {
          return function() {
              that.vcycles = T4;
              that.cpu_cycles = 4;
              var w8 = that.l();
              that.set_l(that.e());
              that.set_e(w8);
              w8 = that.h();
              that.set_h(that.d());
              that.set_d(w8);
          };
        })(this);

    handler[0xEE] =       /* xri data8 */
        (function(that) {
          return function() {
              that.vcycles = T43;
              that.cpu_cycles = 7;
              that.xra_im8(that.next_pc_byte());
          };
        })(this);

    // di/ei, 1111c011
    // c - 0 (di), 1 (ei)
    handler[0xF3] =       /* di */
    handler[0xFB] =       /* ei */
        (function(that) {
          return function(opcode) {
              that.vcycles = T4;
              that.cpu_cycles = 4;
              if ((opcode & 0x08) != 0) {
                that.iff_pending = 2;
              } else {
                that.iff = false;
                that.io.interrupt(false);
              }
          };
        })(this);

    handler[0xF6] =       /* ori data8 */
        (function(that) {
          return function() {
              that.vcycles = T43;
              that.cpu_cycles = 7;
              that.ora_im8(that.next_pc_byte());
          };
        })(this);

    handler[0xF9] =       /* sphl */
        (function(that) {
          return function() {
              that.vcycles = T5;
              that.cpu_cycles = 5;
              that.sp = that.hl();
          };
        })(this);

    handler[0xFE] =       /* cpi data8 */
        (function(that) {
          return function() {
              that.vcycles = T43;
              that.cpu_cycles = 7;
              that.cmp_im8(that.next_pc_byte());
          };
        })(this);

      // mov, 0x40, 01dddsss
      // ddd, sss - b, c, d, e, h, l, m, a
      //            0  1  2  3  4  5  6  7
      handler[0x40] =       /* mov b, b */
      handler[0x41] =       /* mov b, c */
      handler[0x42] =       /* mov b, d */
      handler[0x43] =       /* mov b, e */
      handler[0x44] =       /* mov b, h */
      handler[0x45] =       /* mov b, l */
      handler[0x46] =       /* mov b, m */
      handler[0x47] =       /* mov b, a */
      handler[0x48] =       /* mov c, b */
      handler[0x49] =       /* mov c, c */
      handler[0x4A] =       /* mov c, d */
      handler[0x4B] =       /* mov c, e */
      handler[0x4C] =       /* mov c, h */
      handler[0x4D] =       /* mov c, l */
      handler[0x4E] =       /* mov c, m */
      handler[0x4F] =       /* mov c, a */
      handler[0x50] =       /* mov d, b */
      handler[0x51] =       /* mov d, c */
      handler[0x52] =       /* mov d, d */
      handler[0x53] =       /* mov d, e */
      handler[0x54] =       /* mov d, h */
      handler[0x55] =       /* mov d, l */
      handler[0x56] =       /* mov d, m */
      handler[0x57] =       /* mov d, a */
      handler[0x58] =       /* mov e, b */
      handler[0x59] =       /* mov e, c */
      handler[0x5A] =       /* mov e, d */
      handler[0x5B] =       /* mov e, e */
      handler[0x5C] =       /* mov e, h */
      handler[0x5D] =       /* mov e, l */
      handler[0x5E] =       /* mov e, m */
      handler[0x5F] =       /* mov e, a */
      handler[0x60] =       /* mov h, b */
      handler[0x61] =       /* mov h, c */
      handler[0x62] =       /* mov h, d */
      handler[0x63] =       /* mov h, e */
      handler[0x64] =       /* mov h, h */
      handler[0x65] =       /* mov h, l */
      handler[0x66] =       /* mov h, m */
      handler[0x67] =       /* mov h, a */
      handler[0x68] =       /* mov l, b */
      handler[0x69] =       /* mov l, c */
      handler[0x6A] =       /* mov l, d */
      handler[0x6B] =       /* mov l, e */
      handler[0x6C] =       /* mov l, h */
      handler[0x6D] =       /* mov l, l */
      handler[0x6E] =       /* mov l, m */
      handler[0x6F] =       /* mov l, a */
      handler[0x70] =       /* mov m, b */
      handler[0x71] =       /* mov m, c */
      handler[0x72] =       /* mov m, d */
      handler[0x73] =       /* mov m, e */
      handler[0x74] =       /* mov m, h */
      handler[0x75] =       /* mov m, l */
      handler[0x77] =       /* mov m, a */
      handler[0x78] =       /* mov a, b */
      handler[0x79] =       /* mov a, c */
      handler[0x7A] =       /* mov a, d */
      handler[0x7B] =       /* mov a, e */
      handler[0x7C] =       /* mov a, h */
      handler[0x7D] =       /* mov a, l */
      handler[0x7E] =       /* mov a, m */
      handler[0x7F] =       /* mov a, a */
        (function(that) {
            return function(opcode) {
                const src = opcode & 7;
                const dst = (opcode >> 3) & 7;
                if (src == 6 || dst == 6) {
                    that.vcycles = T43;
                    that.cpu_cycles = 7;
                } else {
                    that.vcycles = T5;
                    that.cpu_cycles = (src == 6 || dst == 6 ? 7 : 5);
                }
                that.set_reg(dst, that.reg(src));
            };
        })(this);

    this.handler = handler;
};

I8080.prototype.execute = function(opcode) {
    this.last_opcode = opcode;
    const h = this.handler[opcode];
    if (h) {
        h(opcode);
    } else {
        alert("Oops! Unhandled opcode " + opcode.toString(16));
    }
    if (this.iff_pending !== 0) {
        if (--this.iff_pending === 0) {
          this.iff = true;
          this.io.interrupt(true);
        }
    }
    return this.cpu_cycles;
}

I8080.prototype.instruction = function() {
  return this.execute(this.next_pc_byte());
}

I8080.prototype.jump = function(addr) {
  this.pc = addr & 0xffff;
}

