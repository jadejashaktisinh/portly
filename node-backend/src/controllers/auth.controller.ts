import type { Request, Response } from 'express';
import bcrypt from 'bcryptjs';
import jwt from 'jsonwebtoken';
import { randomUUID } from 'crypto';
import db from '../providers/db.js';

const JWT_SECRET = process.env.JWT_SECRET!;

export async function signup(req: Request, res: Response) {
  const { email, password } = req.body as { email: string; password: string };

  const hashed = await bcrypt.hash(password, 10);
  const apiKey = randomUUID();

  const user = await db.user.create({
    data: { email, password: hashed, apiKey },
    select: { id: true, email: true, apiKey: true },
  });

  const token = jwt.sign({ userId: user.id }, JWT_SECRET, { expiresIn: '7d' });
  res.status(201).json({ token, user });
}

export async function login(req: Request, res: Response) {
  const { email, password } = req.body as { email: string; password: string };

  const user = await db.user.findUnique({ where: { email } });
  if (!user || !(await bcrypt.compare(password, user.password))) {
    res.status(401).json({ error: 'Invalid credentials' });
    return;
  }

  const token = jwt.sign({ userId: user.id }, JWT_SECRET, { expiresIn: '7d' });
  res.json({ token, user: { id: user.id, email: user.email, apiKey: user.apiKey } });
}
