import express, { type Request, type Response, type NextFunction } from 'express';
import authRoutes from './routes/auth.routes.js';
import eventRoutes from './routes/event.routes.js';
import { config } from './config.js';

const app = express();

app.use(express.json());
app.get('/healthz', (_req, res) => {
	res.json({ status: 'ok' });
});
app.use('/auth', authRoutes);
app.use('/events', eventRoutes);

app.use((err: unknown, _req: Request, res: Response, _next: NextFunction) => {
	console.error(err);
	res.status(500).json({ error: 'Internal server error' });
});

app.listen(config.port, () => {
	console.log(`API listening on port ${config.port}`);
});
