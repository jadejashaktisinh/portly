import express, { type Request, type Response, type NextFunction } from 'express';
import authRoutes from './routes/auth.routes.js';
import eventRoutes from './routes/event.routes.js';

const app = express();
const port = process.env.PORT ?? 3000;

app.use(express.json());
app.use('/auth', authRoutes);
app.use('/events', eventRoutes);

app.use((err: unknown, _req: Request, res: Response, _next: NextFunction) => {
  console.error(err);
  res.status(500).json({ error: 'Internal server error' });
});

app.listen(port, () => {
  console.log(`Server running on http://localhost:${port}`);
});
