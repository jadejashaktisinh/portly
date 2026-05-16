import express from 'express';
import authRoutes from './routes/auth.routes.js';

const app = express();
const port = process.env.PORT ?? 3000;

app.use(express.json());

app.use('/auth', authRoutes);

app.listen(port, () => {
  console.log(`Server running on http://localhost:${port}`);
});
