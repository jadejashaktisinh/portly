import { Router } from 'express';
import { handleWebhook } from '../controllers/webhook.controller.js';

const router = Router();

router.post('/webhook', handleWebhook);

export default router;
