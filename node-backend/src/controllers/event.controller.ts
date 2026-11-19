import type { Request, Response } from "express";
import db from "../providers/db.js";
import type { AuthRequest } from "../middleware/auth.middleware.js";

const eventSelect = {
	id: true,
	requestId: true,
	method: true,
	host: true,
	slug: true,
	path: true,
	headers: true,
	body: true,
	statusCode: true,
	durationMs: true,
	error: true,
	createdAt: true,
} as const;

function paging(req: Request) {
	const limit = Math.min(Math.max(Number.parseInt(String(req.query.limit ?? "50"), 10) || 50, 1), 100);
	const offset = Math.max(Number.parseInt(String(req.query.offset ?? "0"), 10) || 0, 0);
	return { limit, offset };
}

export async function listEvents(req: AuthRequest, res: Response) {
	const userId = req.userId!;
	const { limit, offset } = paging(req);
	const [events, total] = await Promise.all([
		db.webhookEvent.findMany({
			where: { userId },
			orderBy: { createdAt: "desc" },
			take: limit,
			skip: offset,
			select: eventSelect,
		}),
		db.webhookEvent.count({ where: { userId } }),
	]);
	res.json({ events, total, limit, offset });
}

export async function getEvent(req: AuthRequest, res: Response) {
	const rawId = req.params.id;
	const id = Array.isArray(rawId) ? rawId[0] : rawId;
	if (!id) {
		res.status(400).json({ error: "Event id is required" });
		return;
	}

	const event = await db.webhookEvent.findFirst({
		where: { id, userId: req.userId! },
		select: eventSelect,
	});
	if (!event) {
		res.status(404).json({ error: "Event not found" });
		return;
	}
	res.json({ event });
}
