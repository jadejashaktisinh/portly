import type { Request, Response } from 'express';
import proxy from '../providers/proxy.js';


export function handleWebhook(req: Request, res: Response) {
    // Handle the webhook logic here
    console.log('id ->', process.env.RELAY_SERVER_URL + '/send?id=' + req.query.id);
    proxy.web(req, res, {
        target: process.env.RELAY_SERVER_URL + '/send?id=' + req.query.id,
        ignorePath: true,
        toProxy: true,
    }, (err) => {
        console.error('Error proxying webhook:', err);
        res.status(500).json({ message: 'Error processing webhook' });
    });

}