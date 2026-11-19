import type { Request, Response } from "express";
import bcrypt from "bcryptjs";
import jwt from "jsonwebtoken";
import { Prisma } from "@prisma/client";
import { randomUUID } from "crypto";
import db from "../providers/db.js";
import { config } from "../config.js";
import { validateLoginInput, validateSignupInput } from "../validation.js";
import type { AuthRequest } from "../middleware/auth.middleware.js";

const JWT_SECRET = config.jwtSecret;

function isUniqueViolation(error: unknown): boolean {
	return error instanceof Prisma.PrismaClientKnownRequestError && error.code === "P2002";
}

export async function signup(req: Request, res: Response) {
	const validationError = validateSignupInput(req.body);
	if (validationError) {
		res.status(400).json({ error: validationError });
		return;
	}

	const { email, password, slug } = req.body as {
		email: string;
		password: string;
		slug: string;
	};

	const hashed = await bcrypt.hash(password, 10);
	const apiKey = randomUUID();

	try {
		const user = await db.user.create({
			data: { email, password: hashed, apiKey, slug },
			select: { id: true, email: true, apiKey: true, slug: true, createdAt: true },
		});

		const token = jwt.sign({ userId: user.id }, JWT_SECRET, { expiresIn: "7d" });
		res.status(201).json({ token, user });
	} catch (error) {
		if (isUniqueViolation(error)) {
			res.status(409).json({ error: "Email or slug is already in use" });
			return;
		}
		throw error;
	}
}

export async function login(req: Request, res: Response) {
	const validationError = validateLoginInput(req.body);
	if (validationError) {
		res.status(400).json({ error: validationError });
		return;
	}

	const { email, password } = req.body as { email: string; password: string };

	const user = await db.user.findUnique({ where: { email } });
	if (!user || !(await bcrypt.compare(password, user.password))) {
		res.status(401).json({ error: "Invalid credentials" });
		return;
	}

	const token = jwt.sign({ userId: user.id }, JWT_SECRET, { expiresIn: "7d" });
	res.json({
		token,
		user: { id: user.id, email: user.email, apiKey: user.apiKey },
	});
}

export async function me(req: AuthRequest, res: Response) {
	const user = await db.user.findUnique({
		where: { id: req.userId! },
		select: { id: true, email: true, apiKey: true, slug: true, createdAt: true },
	});
	if (!user) {
		res.status(401).json({ error: "Invalid token" });
		return;
	}
	res.json({ user });
}
