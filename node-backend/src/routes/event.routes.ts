import { Router } from 'express';
import { listEvents, getEvent } from '../controllers/event.controller.js';
import { authenticate } from '../middleware/auth.middleware.js';

const router = Router();
router.use(authenticate);
router.get('/', listEvents);
router.get('/:id', getEvent);

export default router;
